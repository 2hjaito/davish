package nginx

import (
	"context"
	"os"
	"path"
	"strings"

	"davish/internal/apperr"
	"davish/internal/shellx"
)

// TogglePlan is what turning a site on or off runs, for the confirmation.
type TogglePlan struct {
	Name    string `json:"name"`
	Enable  bool   `json:"enable"`
	Command string `json:"command"`
	Sudo    string `json:"sudo"`
}

// CheckName rejects anything that is not a plain file name.
func CheckName(name string) error {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\x00\n") || strings.HasPrefix(name, "-") {
		return apperr.New(apperr.InvalidInput, "not a site name")
	}
	return nil
}

// PlanToggle describes turning the site name in sites-available on (a
// link in sites-enabled) or off (removing that link). Only a link is ever
// removed, never a file.
func (s *Service) PlanToggle(_ context.Context, serverID, name string, enable bool) (TogglePlan, error) {
	if err := CheckName(name); err != nil {
		return TogglePlan{}, err
	}
	sc, err := s.conns.SFTP(serverID)
	if err != nil {
		return TogglePlan{}, err
	}
	avail, link := path.Join(DirAvailable, name), path.Join(DirEnabled, name)
	if _, err := sc.Stat(avail); err != nil {
		return TogglePlan{}, apperr.New(apperr.NotFound, "no such site in sites-available")
	}
	fi, lerr := sc.Lstat(link)
	var cmd shellx.Command
	switch {
	case enable && lerr == nil:
		return TogglePlan{}, apperr.New(apperr.FSExists, "the site is already in sites-enabled")
	case enable:
		cmd = shellx.Cmd("ln", "-s", "--", avail, link)
	case lerr != nil:
		return TogglePlan{}, apperr.New(apperr.NotFound, "the site is not enabled")
	case fi.Mode()&os.ModeSymlink == 0:
		return TogglePlan{}, apperr.New(apperr.InvalidInput, "sites-enabled holds a file, not a link")
	default:
		cmd = shellx.Cmd("rm", "--", link)
	}
	return TogglePlan{Name: name, Enable: enable, Command: cmd.String(), Sudo: shellx.Cmd("sudo", cmd.Args()...).String()}, nil
}

// Toggle runs command, which must be the one PlanToggle gives now.
func (s *Service) Toggle(ctx context.Context, serverID, name string, enable bool, command string) error {
	plan, err := s.PlanToggle(ctx, serverID, name, enable)
	if err != nil {
		return err
	}
	if plan.Command != command {
		return apperr.New(apperr.FSStale, "the site changed since the change was confirmed")
	}
	res, err := s.run.RunLine(ctx, serverID, app, command)
	if err != nil {
		return err
	}
	if res.Exit != 0 {
		if strings.Contains(string(res.Stderr), "Permission denied") || strings.Contains(string(res.Stderr), "Operation not permitted") {
			return apperr.New(apperr.CmdNeedsRoot, "sites-enabled is root's")
		}
		return apperr.New(apperr.CmdFailed, "command failed")
	}
	return nil
}

var testCmd = shellx.Cmd("nginx", "-t")

// Test runs nginx -t.
func (s *Service) Test(ctx context.Context, serverID string) (TestResult, error) {
	res, err := s.run.Run(ctx, serverID, app, testCmd)
	if err != nil {
		return TestResult{}, err
	}
	if res.Exit == 127 {
		return TestResult{}, apperr.New(apperr.CmdMissing, "nginx is not installed on the server")
	}
	return parseTest(string(res.Stderr), res.Exit), nil
}

// ReloadResult is what reloading did. Reload runs nginx -t first and
// stops there if it fails.
type ReloadResult struct {
	Test     TestResult `json:"test"`
	Reloaded bool       `json:"reloaded"`
	// NeedsRoot is set when the test or the reload needs root; Sudo then
	// does both as root, for a terminal.
	NeedsRoot bool   `json:"needsRoot"`
	Sudo      string `json:"sudo"`
}

// ReloadCommand is exactly what Reload runs: the reload only happens if
// the test passes.
var ReloadCommand = testCmd.String() + " && " + shellx.Cmd("systemctl", "reload", "nginx").String()

var reloadSudo = shellx.Cmd("sudo", "nginx", "-t").String() + " && " + shellx.Cmd("sudo", "systemctl", "reload", "nginx").String()

// Reload tests the configuration and, if it passes, reloads nginx.
func (s *Service) Reload(ctx context.Context, serverID string) (ReloadResult, error) {
	res, err := s.run.RunLine(ctx, serverID, app, ReloadCommand)
	if err != nil {
		return ReloadResult{}, err
	}
	out := string(res.Stderr)
	r := ReloadResult{Sudo: reloadSudo}
	if res.Exit == 0 {
		r.Test = parseTest(out, 0)
		r.Reloaded = true
		return r, nil
	}
	if res.Exit == 127 {
		return ReloadResult{}, apperr.New(apperr.CmdMissing, "nginx is not installed on the server")
	}
	if strings.Contains(out, "test is successful") {
		// The test passed; systemctl refused or failed.
		r.Test = parseTest(out, 0)
		if strings.Contains(out, "Interactive authentication required") || strings.Contains(out, "Access denied") {
			r.NeedsRoot = true
			return r, nil
		}
		return r, apperr.New(apperr.CmdFailed, "nginx did not reload")
	}
	r.Test = parseTest(out, res.Exit)
	r.NeedsRoot = r.Test.NeedsRoot
	return r, nil
}
