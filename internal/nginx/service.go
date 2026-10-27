package nginx

import (
	"context"
	"io"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/pkg/sftp"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/shellx"
)

// app is the name commands are recorded under in the audit log.
const app = "nginx"

// Where Debian-style nginx keeps its sites.
const (
	DirAvailable = "/etc/nginx/sites-available"
	DirEnabled   = "/etc/nginx/sites-enabled"
	DirConfD     = "/etc/nginx/conf.d"
)

// Runner runs a command on a server and records it in the audit log.
type Runner interface {
	Run(ctx context.Context, serverID, app string, cmd shellx.Command) (audit.Result, error)
	RunLine(ctx context.Context, serverID, app, line string) (audit.Result, error)
}

// Conns hands out a server's shared SFTP client, to read the config.
type Conns interface {
	SFTP(serverID string) (*sftp.Client, error)
}

// Emitter pushes events to the frontend.
type Emitter interface {
	Emit(name string, payload any)
}

// Site kinds.
const (
	KindSite = "site" // sites-available, turned on by a link in sites-enabled
	KindConf = "conf" // conf.d/*.conf, always on
)

// Site is one config file of a site.
type Site struct {
	Name string `json:"name"`
	// Path is the file to open in the Editor.
	Path    string `json:"path"`
	Kind    string `json:"kind"`
	Enabled bool   `json:"enabled"`
	// Toggleable is false when on/off is not a link: conf.d files, and
	// files put in sites-enabled directly.
	Toggleable bool     `json:"toggleable"`
	Servers    []Server `json:"servers"`
	Upstreams  []string `json:"upstreams"`
	Certs      []Cert   `json:"certs"`
	// Error is set when the file could not be read.
	Error string `json:"error,omitempty"`
}

// Service manages nginx on each server.
type Service struct {
	run   Runner
	conns Conns
	certs *certWatch
}

// New creates the service.
func New(run Runner, conns Conns, emit Emitter) *Service {
	return &Service{run: run, conns: conns, certs: newCertWatch(emit)}
}

// List returns the sites of sites-available, sites-enabled and conf.d,
// sorted by name, with the certificates of enabled sites.
func (s *Service) List(_ context.Context, serverID string) ([]Site, error) {
	sc, err := s.conns.SFTP(serverID)
	if err != nil {
		return nil, err
	}
	if _, err := sc.Stat("/etc/nginx"); err != nil {
		return nil, apperr.New(apperr.CmdMissing, "nginx is not installed on the server")
	}
	sites := map[string]*Site{}
	for _, name := range dirNames(sc, DirAvailable) {
		sites[name] = &Site{Name: name, Path: path.Join(DirAvailable, name), Kind: KindSite, Toggleable: true}
	}
	for _, name := range dirNames(sc, DirEnabled) {
		p := path.Join(DirEnabled, name)
		fi, err := sc.Lstat(p)
		if err != nil {
			continue
		}
		site := sites[name]
		if fi.Mode()&os.ModeSymlink != 0 {
			target, err := sc.ReadLink(p)
			if err == nil && site != nil && resolve(DirEnabled, target) == site.Path {
				site.Enabled = true
				continue
			}
			// A link to somewhere else: shown as its own site, read
			// through the link.
		}
		sites["enabled:"+name] = &Site{Name: name, Path: p, Kind: KindSite, Enabled: true}
	}
	for _, name := range dirNames(sc, DirConfD) {
		if strings.HasSuffix(name, ".conf") {
			sites["conf:"+name] = &Site{Name: name, Path: path.Join(DirConfD, name), Kind: KindConf, Enabled: true}
		}
	}

	out := make([]Site, 0, len(sites))
	for _, site := range sites {
		site.Servers, site.Upstreams, site.Certs = []Server{}, []string{}, []Cert{}
		if text, err := readFile(sc, site.Path); err != nil {
			site.Error = "unreadable"
		} else {
			p := Parse(text)
			site.Servers, site.Upstreams = p.Servers, p.Upstreams
		}
		if site.Enabled {
			site.Certs = s.certs.check(sc, serverID, site.Servers)
		}
		out = append(out, *site)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Path < out[j].Path
	})
	return out, nil
}

// CheckCerts reads the certificates of enabled sites and notifies the ones
// that expire soon. It is List without the result, for when a server
// connects.
func (s *Service) CheckCerts(ctx context.Context, serverID string) error {
	_, err := s.List(ctx, serverID)
	if err != nil && isMissing(err) {
		return nil
	}
	return err
}

func isMissing(err error) bool {
	ae, ok := err.(*apperr.Error) //nolint:errorlint // List returns *apperr.Error unwrapped
	return ok && ae.Code == apperr.CmdMissing
}

func dirNames(sc *sftp.Client, dir string) []string {
	entries, err := sc.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if n := e.Name(); !strings.HasPrefix(n, ".") && !strings.HasSuffix(n, "~") && !strings.Contains(n, ".dpkg-") {
			if !e.IsDir() {
				names = append(names, n)
			}
		}
	}
	return names
}

// resolve turns a link target into an absolute path from dir.
func resolve(dir, target string) string {
	if path.IsAbs(target) {
		return path.Clean(target)
	}
	return path.Join(dir, target)
}

func readFile(sc *sftp.Client, p string) (string, error) {
	f, err := sc.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close() //nolint:errcheck // read only
	data, err := io.ReadAll(io.LimitReader(f, 1<<20))
	return string(data), err
}
