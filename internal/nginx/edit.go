package nginx

import (
	"context"
	"io"
	"path"
	"regexp"
	"slices"
	"strings"
	"time"

	"davish/internal/apperr"
	"davish/internal/shellx"
)

// SiteEdit is a quick change to an existing site. Empty fields are left
// as they are.
type SiteEdit struct {
	// AddNames are added to every server_name of the file.
	AddNames []string `json:"addNames"`
	// Upstream replaces the target of proxy_pass.
	Upstream string `json:"upstream"`
	// Gzip is "on", "off" or "" to keep.
	Gzip string `json:"gzip"`
	// MaxBody sets client_max_body_size, such as 50m.
	MaxBody string `json:"maxBody"`
}

// EditPlan is what a quick change writes and runs, for the preview.
type EditPlan struct {
	Path string `json:"path"`
	// Backup is where the current file is copied first; it is put back if
	// nginx -t fails.
	Backup  string `json:"backup"`
	Before  string `json:"before"`
	After   string `json:"after"`
	Command string `json:"command"`
	Sudo    string `json:"sudo"`
}

var (
	maxBodyRe    = regexp.MustCompile(`^\d{1,6}[kKmMgG]?$`)
	serverNameRe = regexp.MustCompile(`^(\s*)server_name\s+([^;#]*);`)
	proxyPassRe  = regexp.MustCompile(`^(\s*)proxy_pass\s+([^;\s]+)\s*;`)
	backupRe     = regexp.MustCompile(`\.bak-\d{8}T\d{6}Z$`)
)

func (e SiteEdit) check() (SiteEdit, error) {
	bad := func(msg string) (SiteEdit, error) { return e, apperr.New(apperr.InvalidInput, msg) }
	var names []string
	for _, n := range e.AddNames {
		n = strings.ToLower(strings.TrimSpace(n))
		if n == "" || slices.Contains(names, n) {
			continue
		}
		if !hostName.MatchString(n) {
			return bad("not a domain name")
		}
		names = append(names, n)
	}
	e.AddNames = names
	if e.Upstream = strings.TrimSpace(e.Upstream); e.Upstream != "" {
		if !upstreamRe.MatchString(e.Upstream) {
			return bad("not an address to proxy to")
		}
		if !strings.Contains(e.Upstream, "://") {
			e.Upstream = "http://" + e.Upstream
		}
	}
	if e.Gzip != "" && e.Gzip != "on" && e.Gzip != "off" {
		return bad("gzip is on or off")
	}
	if e.MaxBody = strings.TrimSpace(e.MaxBody); e.MaxBody != "" && !maxBodyRe.MatchString(e.MaxBody) {
		return bad("not a size, such as 50m")
	}
	if len(e.AddNames) == 0 && e.Upstream == "" && e.Gzip == "" && e.MaxBody == "" {
		return bad("nothing to change")
	}
	return e, nil
}

// applyEdit changes config line by line. Directives it sets are replaced
// where they are, or added after each server_name.
func applyEdit(config string, e SiteEdit) (string, error) {
	lines := strings.Split(config, "\n")
	targets := map[string]bool{}
	for _, l := range lines {
		if m := proxyPassRe.FindStringSubmatch(l); m != nil {
			targets[m[2]] = true
		}
	}
	if e.Upstream != "" && len(targets) != 1 {
		return "", apperr.New(apperr.InvalidInput, "the site must proxy to exactly one address to change it here")
	}
	set := map[string]string{}
	if e.Gzip != "" {
		set["gzip"] = e.Gzip
	}
	if e.MaxBody != "" {
		set["client_max_body_size"] = e.MaxBody
	}
	found := map[string]bool{}
	for i, l := range lines {
		for d, v := range set {
			re := regexp.MustCompile(`^(\s*)` + d + `\s+[^;]*;`)
			if m := re.FindStringSubmatch(l); m != nil {
				lines[i] = m[1] + d + " " + v + ";" + l[len(m[0]):]
				found[d] = true
			}
		}
		if m := proxyPassRe.FindStringSubmatch(l); m != nil && e.Upstream != "" {
			lines[i] = m[1] + "proxy_pass " + e.Upstream + ";" + l[len(m[0]):]
		}
	}
	var out []string
	names := 0
	for _, l := range lines {
		m := serverNameRe.FindStringSubmatch(l)
		if m == nil {
			out = append(out, l)
			continue
		}
		names++
		have := strings.Fields(m[2])
		add := slices.DeleteFunc(slices.Clone(e.AddNames), func(n string) bool { return slices.Contains(have, n) })
		if len(add) > 0 {
			l = m[1] + "server_name " + strings.Join(append(have, add...), " ") + ";" + l[len(m[0]):]
		}
		out = append(out, l)
		for _, d := range []string{"gzip", "client_max_body_size"} {
			if v, ok := set[d]; ok && !found[d] {
				out = append(out, m[1]+d+" "+v+";")
			}
		}
	}
	if names == 0 && (len(e.AddNames) > 0 || len(set) > len(found)) {
		return "", apperr.New(apperr.InvalidInput, "the file has no server_name to change")
	}
	return strings.Join(out, "\n"), nil
}

// PlanEdit describes a quick change to the site file at sitePath. backup
// is the backup name of an earlier plan, to plan the same command again;
// empty makes a new one.
func (s *Service) PlanEdit(_ context.Context, serverID, sitePath string, e SiteEdit, backup string) (EditPlan, error) {
	if err := checkSitePath(sitePath); err != nil {
		return EditPlan{}, err
	}
	if path.Dir(sitePath) == DirEnabled {
		// A backup next to it would be loaded by nginx too.
		return EditPlan{}, apperr.New(apperr.InvalidInput, "files directly in sites-enabled are changed in the Editor")
	}
	if backup == "" {
		backup = sitePath + ".bak-" + time.Now().UTC().Format("20060102T150405Z")
	} else if !strings.HasPrefix(backup, sitePath+".bak-") || !backupRe.MatchString(backup) {
		return EditPlan{}, apperr.New(apperr.InvalidInput, "not a backup name")
	}
	e, err := e.check()
	if err != nil {
		return EditPlan{}, err
	}
	sc, err := s.conns.SFTP(serverID)
	if err != nil {
		return EditPlan{}, err
	}
	f, err := sc.Open(sitePath)
	if err != nil {
		return EditPlan{}, apperr.New(apperr.NotFound, "could not read the site file")
	}
	defer func() { _ = f.Close() }()
	b, err := io.ReadAll(io.LimitReader(f, 1<<20))
	if err != nil {
		return EditPlan{}, err
	}
	after, err := applyEdit(string(b), e)
	if err != nil {
		return EditPlan{}, err
	}
	p := EditPlan{Path: sitePath, Backup: backup, Before: string(b), After: after}
	p.Command, p.Sudo = editLine(p, false), editLine(p, true)
	return p, nil
}

// editLine backs the file up, writes the new content and reloads nginx,
// putting the backup back if the test fails.
func editLine(p EditPlan, sudo bool) string {
	as := func(c shellx.Command) string {
		if sudo {
			c = shellx.Cmd("sudo", c.Args()...)
		}
		return c.String()
	}
	return strings.Join([]string{
		as(shellx.Cmd("cp", "-p", "--", p.Path, p.Backup)),
		shellx.Cmd("printf", "%s", p.After).String() + " | " + as(shellx.Cmd("tee", "--", p.Path)) + " > /dev/null",
		"{ " + as(testCmd) + " && " + as(shellx.Cmd("systemctl", "reload", "nginx")) +
			" || { " + as(shellx.Cmd("cp", "-p", "--", p.Backup, p.Path)) + "; false; }; }",
	}, " && ")
}

// Edit runs command, which must be what PlanEdit gives now with backup.
func (s *Service) Edit(ctx context.Context, serverID, sitePath string, e SiteEdit, backup, command string) (CreateResult, error) {
	plan, err := s.PlanEdit(ctx, serverID, sitePath, e, backup)
	if err != nil {
		return CreateResult{}, err
	}
	if plan.Command != command {
		return CreateResult{}, apperr.New(apperr.FSStale, "the site changed since the change was confirmed")
	}
	res, err := s.run.RunLine(ctx, serverID, app, command)
	if err != nil {
		return CreateResult{}, err
	}
	return createResult(string(res.Stderr), res.Exit, plan.Sudo)
}
