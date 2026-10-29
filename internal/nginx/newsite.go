package nginx

import (
	"bytes"
	"context"
	_ "embed"
	"path"
	"regexp"
	"slices"
	"strings"
	"text/template"

	"davish/internal/apperr"
	"davish/internal/shellx"
)

//go:embed templates/site.conf.tmpl
var siteTmpl string

var siteTemplate = template.Must(template.New("site").Funcs(template.FuncMap{"join": strings.Join}).Parse(siteTmpl))

// Site types a new site can be.
const (
	TypeStatic = "static"
	TypeProxy  = "proxy"
	TypePHP    = "php"
)

// NewSite is the form of a new site.
type NewSite struct {
	Names []string `json:"names"`
	Type  string   `json:"type"`
	// Root is the folder served by static and PHP sites.
	Root string `json:"root"`
	// Upstream is where a proxy sends requests: host:port or a URL.
	Upstream  string `json:"upstream"`
	Websocket bool   `json:"websocket"`
	// PHPSocket is the php-fpm socket of a PHP site.
	PHPSocket string `json:"phpSocket"`
}

// SiteOptions are what the new site form offers on a server.
type SiteOptions struct {
	// ConfD is set when the server has no sites-available, so new sites go
	// to conf.d.
	ConfD      bool     `json:"confD"`
	PHPSockets []string `json:"phpSockets"`
}

// CreatePlan is what creating a site writes and runs, for the preview.
type CreatePlan struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Config  string `json:"config"`
	Command string `json:"command"`
	Sudo    string `json:"sudo"`
}

// CreateResult is what creating a site did.
type CreateResult struct {
	// Created is set when the site is written, enabled and nginx reloaded.
	Created bool `json:"created"`
	// Test is nginx -t's verdict when it ran; a failing test leaves the
	// site off.
	Test *TestResult `json:"test,omitempty"`
	// NeedsRoot is set when the server refused the writes; Sudo does it
	// all as root, for a terminal.
	NeedsRoot bool   `json:"needsRoot"`
	Sudo      string `json:"sudo"`
}

var (
	hostName   = regexp.MustCompile(`^(\*\.)?([A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?\.)*[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?$`)
	upstreamRe = regexp.MustCompile(`^(https?://)?[A-Za-z0-9._\-\[\]:]+(:\d{1,5})?(/[A-Za-z0-9._~/\-]*)?$`)
	safePath   = regexp.MustCompile(`^/[A-Za-z0-9._@+/\-]*$`)
)

func (n NewSite) check() (NewSite, error) {
	bad := func(msg string) (NewSite, error) { return n, apperr.New(apperr.InvalidInput, msg) }
	var names []string
	for _, name := range n.Names {
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" || slices.Contains(names, name) {
			continue
		}
		if name != "_" && !hostName.MatchString(name) {
			return bad("not a domain name")
		}
		names = append(names, name)
	}
	if len(names) == 0 || names[0] == "_" {
		return bad("a site needs a domain name")
	}
	n.Names = names
	switch n.Type {
	case TypeStatic, TypePHP:
		n.Root = path.Clean(strings.TrimSpace(n.Root))
		if !safePath.MatchString(n.Root) || n.Root == "/" {
			return bad("not a folder for the site")
		}
		if n.Type == TypePHP && !safePath.MatchString(n.PHPSocket) {
			return bad("not a php-fpm socket")
		}
	case TypeProxy:
		n.Upstream = strings.TrimSpace(n.Upstream)
		if !upstreamRe.MatchString(n.Upstream) {
			return bad("not an address to proxy to")
		}
		if !strings.Contains(n.Upstream, "://") {
			n.Upstream = "http://" + n.Upstream
		}
	default:
		return bad("unknown site type")
	}
	return n, nil
}

// RenderSite returns the config for a new site.
func RenderSite(n NewSite) (string, error) {
	n, err := n.check()
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	if err := siteTemplate.Execute(&b, n); err != nil {
		return "", err
	}
	return b.String(), nil
}

// Options looks at how the server keeps its sites and which php-fpm
// sockets it has.
func (s *Service) Options(_ context.Context, serverID string) (SiteOptions, error) {
	sc, err := s.conns.SFTP(serverID)
	if err != nil {
		return SiteOptions{}, err
	}
	o := SiteOptions{PHPSockets: []string{}}
	if _, err := sc.Stat(DirAvailable); err != nil {
		o.ConfD = true
	}
	for _, pattern := range []string{"/run/php/*.sock", "/var/run/php/*.sock", "/run/php-fpm/*.sock"} {
		matches, _ := sc.Glob(pattern)
		for _, m := range matches {
			if !slices.Contains(o.PHPSockets, m) && !slices.Contains(o.PHPSockets, strings.Replace(m, "/var/run/", "/run/", 1)) {
				o.PHPSockets = append(o.PHPSockets, m)
			}
		}
	}
	return o, nil
}

// PlanCreate describes writing config as a new site for n, enabling it and
// reloading nginx. The site is turned off again if nginx -t fails.
func (s *Service) PlanCreate(ctx context.Context, serverID string, n NewSite, config string) (CreatePlan, error) {
	n, err := n.check()
	if err != nil {
		return CreatePlan{}, err
	}
	opts, err := s.Options(ctx, serverID)
	if err != nil {
		return CreatePlan{}, err
	}
	sc, err := s.conns.SFTP(serverID)
	if err != nil {
		return CreatePlan{}, err
	}
	p := planCreate(n, config, opts.ConfD)
	if _, err := sc.Stat(p.Path); err == nil {
		return CreatePlan{}, apperr.New(apperr.FSExists, "a site with this name exists")
	}
	return p, nil
}

func planCreate(n NewSite, config string, confD bool) CreatePlan {
	name := n.Names[0]
	if name[0] == '*' {
		name = "wildcard" + name[1:]
	}
	if !strings.HasSuffix(config, "\n") {
		config += "\n"
	}
	p := CreatePlan{Name: name, Config: config}
	var pre, post []shellx.Command
	var undo shellx.Command
	if n.Type == TypeStatic || n.Type == TypePHP {
		pre = append(pre, shellx.Cmd("mkdir", "-p", "--", n.Root))
	}
	if confD {
		p.Name += ".conf"
		p.Path = path.Join(DirConfD, p.Name)
		undo = shellx.Cmd("rm", "--", p.Path)
	} else {
		p.Path = path.Join(DirAvailable, p.Name)
		link := path.Join(DirEnabled, p.Name)
		post = append(post, shellx.Cmd("ln", "-s", "--", p.Path, link))
		undo = shellx.Cmd("rm", "--", link)
	}
	p.Command = createLine(pre, p.Path, post, undo, config, false)
	p.Sudo = createLine(pre, p.Path, post, undo, config, true)
	return p
}

// createLine writes config to file with printf and tee, between the pre
// and post steps, then tests and reloads nginx, running undo if the test
// fails. The config is one quoted argument, so a terminal it is typed into
// waits for Enter after the last line.
func createLine(pre []shellx.Command, file string, post []shellx.Command, undo shellx.Command, config string, sudo bool) string {
	as := func(c shellx.Command) string {
		if sudo {
			c = shellx.Cmd("sudo", c.Args()...)
		}
		return c.String()
	}
	var parts []string
	for _, c := range pre {
		parts = append(parts, as(c))
	}
	parts = append(parts, shellx.Cmd("printf", "%s", config).String()+" | "+as(shellx.Cmd("tee", "--", file))+" > /dev/null")
	for _, c := range post {
		parts = append(parts, as(c))
	}
	parts = append(parts, "{ "+as(testCmd)+" && "+as(shellx.Cmd("systemctl", "reload", "nginx"))+" || { "+as(undo)+"; false; }; }")
	return strings.Join(parts, " && ")
}

// Create runs command, which must be what PlanCreate gives now.
func (s *Service) Create(ctx context.Context, serverID string, n NewSite, config, command string) (CreateResult, error) {
	plan, err := s.PlanCreate(ctx, serverID, n, config)
	if err != nil {
		return CreateResult{}, err
	}
	if plan.Command != command {
		return CreateResult{}, apperr.New(apperr.FSStale, "the site changed since it was confirmed")
	}
	res, err := s.run.RunLine(ctx, serverID, app, command)
	if err != nil {
		return CreateResult{}, err
	}
	return createResult(string(res.Stderr), res.Exit, plan.Sudo)
}

func createResult(out string, exit int, sudo string) (CreateResult, error) {
	r := CreateResult{Sudo: sudo}
	if exit == 0 {
		t := parseTest(out, 0)
		r.Created, r.Test = true, &t
		return r, nil
	}
	if strings.Contains(out, "nginx: configuration file") || strings.Contains(out, "[emerg]") {
		t := parseTest(out, exit)
		r.Test = &t
		if t.NeedsRoot {
			r.NeedsRoot = true
		} else if strings.Contains(out, "test is successful") {
			// The test passed; systemctl refused or failed.
			if strings.Contains(out, "Interactive authentication required") || strings.Contains(out, "Access denied") {
				r.NeedsRoot = true
				return r, nil
			}
			return r, apperr.New(apperr.CmdFailed, "nginx did not reload")
		}
		return r, nil
	}
	if strings.Contains(out, "Permission denied") || strings.Contains(out, "Operation not permitted") {
		r.NeedsRoot = true
		return r, nil
	}
	if exit == 127 {
		return r, apperr.New(apperr.CmdMissing, "nginx is not installed on the server")
	}
	return r, apperr.New(apperr.CmdFailed, "command failed")
}

// PlanCreateFor is PlanCreate without the server: for a plan made ahead,
// such as the deploy wizard's, where the server's layout is passed in.
func PlanCreateFor(n NewSite, config string, confD bool) (CreatePlan, error) {
	n, err := n.check()
	if err != nil {
		return CreatePlan{}, err
	}
	return planCreate(n, config, confD), nil
}
