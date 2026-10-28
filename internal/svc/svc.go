// Package svc lists and controls systemd services through systemctl.
package svc

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"regexp"
	"sort"
	"strings"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/shellx"
)

// app is the name commands are recorded under in the audit log.
const app = "services"

// Runner runs a command on a server and records it in the audit log.
type Runner interface {
	Run(ctx context.Context, serverID, app string, cmd shellx.Command) (audit.Result, error)
	RunLine(ctx context.Context, serverID, app, line string) (audit.Result, error)
}

// Unit is one row of systemctl list-units.
type Unit struct {
	Name        string `json:"name"`
	Load        string `json:"load"`   // loaded, not-found, masked, error
	Active      string `json:"active"` // active, inactive, failed, activating, deactivating, reloading
	Sub         string `json:"sub"`    // running, exited, dead, failed, auto-restart…
	Description string `json:"description"`
}

// Service talks to systemd on each server.
type Service struct {
	run Runner
}

// New creates the service.
func New(run Runner) *Service {
	return &Service{run: run}
}

// List returns the server's service units sorted by name. It asks for JSON
// (systemd 245+) and falls back to the plain table on older systems.
func (s *Service) List(ctx context.Context, serverID string) ([]Unit, error) {
	base := []string{"list-units", "--type=service", "--all", "--no-pager"}
	res, err := s.run.Run(ctx, serverID, app, shellx.Cmd("systemctl", append(base, "--output=json")...))
	if err != nil {
		return nil, err
	}
	if err := checkExit(res); err != nil {
		return nil, err
	}
	units, err := parseJSON(res.Stdout)
	if err != nil {
		// Before 245, --output only applied to journal lines and the
		// table was printed anyway.
		res, err = s.run.Run(ctx, serverID, app, shellx.Cmd("systemctl", append(base, "--plain", "--no-legend")...))
		if err != nil {
			return nil, err
		}
		if err := checkExit(res); err != nil {
			return nil, err
		}
		units = parsePlain(res.Stdout)
	}
	return tidy(units), nil
}

func parseJSON(out []byte) ([]Unit, error) {
	var rows []struct {
		Unit        string `json:"unit"`
		Load        string `json:"load"`
		Active      string `json:"active"`
		Sub         string `json:"sub"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(out, &rows); err != nil {
		return nil, err
	}
	units := make([]Unit, len(rows))
	for i, r := range rows {
		units[i] = Unit{Name: r.Unit, Load: r.Load, Active: r.Active, Sub: r.Sub, Description: r.Description}
	}
	return units, nil
}

// parsePlain reads "UNIT LOAD ACTIVE SUB DESCRIPTION" rows; the
// description is the rest of the line.
func parsePlain(out []byte) []Unit {
	var units []Unit
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		// Without --plain, failed units are marked with a bullet.
		line = strings.TrimSpace(strings.TrimPrefix(line, "●"))
		f := strings.Fields(line)
		if len(f) < 4 {
			continue
		}
		u := Unit{Name: f[0], Load: f[1], Active: f[2], Sub: f[3]}
		if len(f) > 4 {
			u.Description = strings.Join(f[4:], " ")
		}
		units = append(units, u)
	}
	return units
}

// tidy drops units that are only referenced, never installed (a Wants= on
// a package that is not there), unless something of them still runs, and
// sorts by name.
func tidy(units []Unit) []Unit {
	out := units[:0]
	for _, u := range units {
		if !strings.HasSuffix(u.Name, ".service") {
			continue
		}
		if u.Load == "not-found" && u.Active == "inactive" {
			continue
		}
		out = append(out, u)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// unitName matches what systemd allows in a unit name, starting with a
// letter or digit so it can never be read as an option.
var unitName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9:_.@\\-]*\.service$`)

// CheckUnit rejects anything that is not a plain service unit name.
func CheckUnit(unit string) error {
	if len(unit) > 256 || !unitName.MatchString(unit) {
		return apperr.New(apperr.InvalidInput, "not a service unit name")
	}
	return nil
}

// checkExit maps a failed systemctl or journalctl to an error code.
func checkExit(res audit.Result) error {
	if res.Exit == 0 {
		return nil
	}
	if res.Exit == 127 {
		return apperr.New(apperr.CmdMissing, "program not installed on the server")
	}
	msg := string(res.Stderr)
	switch {
	case needsRoot(msg):
		return apperr.New(apperr.CmdNeedsRoot, "the server requires root for this")
	case strings.Contains(msg, "not loaded"), strings.Contains(msg, "not found"):
		return apperr.New(apperr.NotFound, "no such service")
	}
	return apperr.New(apperr.CmdFailed, "command failed")
}

func needsRoot(stderr string) bool {
	for _, s := range []string{
		"Interactive authentication required",
		"Access denied",
		"Permission denied",
		"Operation not permitted",
	} {
		if strings.Contains(stderr, s) {
			return true
		}
	}
	return false
}
