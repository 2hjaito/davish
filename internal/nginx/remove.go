package nginx

import (
	"context"
	"path"
	"regexp"
	"slices"
	"strings"

	"davish/internal/apperr"
	"davish/internal/shellx"
)

// RemovePlan is what deleting a site runs, for the confirmation.
type RemovePlan struct {
	Path string `json:"path"`
	Name string `json:"name"`
	// Enabled tells that nginx still serves the site until it reloads.
	Enabled bool `json:"enabled"`
	// Certs are the Let's Encrypt certificates only this site uses; they
	// are deleted with it when the user asks (WithCerts).
	Certs     []string `json:"certs"`
	WithCerts bool     `json:"withCerts"`
	// SharedCerts are used by other sites too and are always kept.
	SharedCerts []string `json:"sharedCerts"`
	Command     string   `json:"command"`
	Sudo        string   `json:"sudo"`
}

// leCert is a certificate certbot manages: /etc/letsencrypt/live/<name>/….
var leCert = regexp.MustCompile(`^/etc/letsencrypt/live/([A-Za-z0-9][A-Za-z0-9._-]*)/[^/]+$`)

// PlanRemove describes deleting the site file at sitePath, with the link
// that enables it, and with withCerts the certificates only it uses.
func (s *Service) PlanRemove(ctx context.Context, serverID, sitePath string, withCerts bool) (RemovePlan, error) {
	if err := checkSitePath(sitePath); err != nil {
		return RemovePlan{}, err
	}
	sites, err := s.List(ctx, serverID)
	if err != nil {
		return RemovePlan{}, err
	}
	return planRemove(sites, sitePath, withCerts)
}

func checkSitePath(p string) error {
	dir, name := path.Split(p)
	dir = path.Clean(dir)
	if CheckName(name) != nil || (dir != DirAvailable && dir != DirEnabled && dir != DirConfD) {
		return apperr.New(apperr.InvalidInput, "not a site file")
	}
	return nil
}

func planRemove(sites []Site, sitePath string, withCerts bool) (RemovePlan, error) {
	i := slices.IndexFunc(sites, func(x Site) bool { return x.Path == sitePath })
	if i < 0 {
		return RemovePlan{}, apperr.New(apperr.NotFound, "no such site")
	}
	site := sites[i]
	plan := RemovePlan{Path: site.Path, Name: site.Name, Enabled: site.Enabled, Certs: []string{}, SharedCerts: []string{}}

	var steps []shellx.Command
	if site.Toggleable && site.Enabled {
		steps = append(steps, shellx.Cmd("rm", "--", path.Join(DirEnabled, site.Name)))
	}
	steps = append(steps, shellx.Cmd("rm", "--", site.Path))

	others := map[string]bool{}
	for j, other := range sites {
		if j != i {
			for _, n := range certNames(other) {
				others[n] = true
			}
		}
	}
	for _, n := range certNames(site) {
		if others[n] {
			plan.SharedCerts = append(plan.SharedCerts, n)
		} else {
			plan.Certs = append(plan.Certs, n)
		}
	}
	if withCerts && len(plan.Certs) > 0 {
		plan.WithCerts = true
		for _, n := range plan.Certs {
			steps = append(steps, shellx.Cmd("certbot", "delete", "--non-interactive", "--cert-name", n))
		}
	}
	plan.Command, plan.Sudo = chain(steps, false), chain(steps, true)
	return plan, nil
}

// certNames are the Let's Encrypt certificate names a site refers to.
func certNames(site Site) []string {
	var names []string
	for _, srv := range site.Servers {
		for _, c := range srv.Certs {
			if m := leCert.FindStringSubmatch(c); m != nil && !slices.Contains(names, m[1]) {
				names = append(names, m[1])
			}
		}
	}
	return names
}

// chain joins steps with &&, each run as root when sudo is set.
func chain(steps []shellx.Command, sudo bool) string {
	parts := make([]string, len(steps))
	for i, c := range steps {
		if sudo {
			c = shellx.Cmd("sudo", c.Args()...)
		}
		parts[i] = c.String()
	}
	return strings.Join(parts, " && ")
}

// Remove runs command, which must be what PlanRemove gives now.
func (s *Service) Remove(ctx context.Context, serverID, sitePath string, withCerts bool, command string) error {
	plan, err := s.PlanRemove(ctx, serverID, sitePath, withCerts)
	if err != nil {
		return err
	}
	if plan.Command != command {
		return apperr.New(apperr.FSStale, "the site changed since the removal was confirmed")
	}
	res, err := s.run.RunLine(ctx, serverID, app, command)
	if err != nil {
		return err
	}
	if res.Exit == 0 {
		return nil
	}
	if res.Exit == 127 {
		return apperr.New(apperr.CmdMissing, "certbot is not installed on the server")
	}
	out := string(res.Stderr)
	if strings.Contains(out, "Permission denied") || strings.Contains(out, "Operation not permitted") ||
		strings.Contains(out, "requires root") || strings.Contains(out, "must be root") {
		return apperr.New(apperr.CmdNeedsRoot, "the site files are root's")
	}
	return apperr.New(apperr.CmdFailed, "command failed")
}
