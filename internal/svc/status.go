package svc

import (
	"bufio"
	"bytes"
	"context"
	"strconv"
	"strings"

	"davish/internal/shellx"
)

// Status is the detail of one unit, from systemctl show.
type Status struct {
	Unit        string `json:"unit"`
	Description string `json:"description"`
	Load        string `json:"load"`
	Active      string `json:"active"`
	Sub         string `json:"sub"`
	// Enabled is the unit file state: enabled, disabled, static, masked…
	Enabled string `json:"enabled"`
	// ActiveFor is how long the unit has been active, in seconds; 0 when
	// it is not active.
	ActiveFor int64  `json:"activeFor"`
	MainPID   int    `json:"mainPID"`
	Memory    int64  `json:"memory"` // bytes, -1 when not tracked
	Restarts  int    `json:"restarts"`
	Result    string `json:"result"` // success, exit-code, signal, timeout…
	// File is the unit file, to open in the Editor.
	File string `json:"file"`
}

var statusProps = []string{
	"Description", "LoadState", "ActiveState", "SubState", "UnitFileState",
	"ActiveEnterTimestampMonotonic", "MainPID", "MemoryCurrent", "NRestarts",
	"Result", "FragmentPath",
}

// Status returns the detail of unit. How long it has run comes from the
// monotonic start time and the server's uptime: the wall-clock property is
// printed with a time zone abbreviation that cannot be parsed reliably.
func (s *Service) Status(ctx context.Context, serverID, unit string) (Status, error) {
	if err := CheckUnit(unit); err != nil {
		return Status{}, err
	}
	args := []string{"show", unit, "--no-pager", "-p", strings.Join(statusProps, ",")}
	res, err := s.run.Run(ctx, serverID, app, shellx.Cmd("systemctl", args...))
	if err != nil {
		return Status{}, err
	}
	if err := checkExit(res); err != nil {
		return Status{}, err
	}
	up, err := s.run.Run(ctx, serverID, app, shellx.Cmd("cat", "/proc/uptime"))
	if err != nil {
		return Status{}, err
	}
	return parseStatus(unit, res.Stdout, up.Stdout), nil
}

// parseStatus reads systemctl show's key=value lines; uptime is the
// content of /proc/uptime.
func parseStatus(unit string, show, uptime []byte) Status {
	p := map[string]string{}
	sc := bufio.NewScanner(bytes.NewReader(show))
	for sc.Scan() {
		if k, v, ok := strings.Cut(sc.Text(), "="); ok {
			p[k] = v
		}
	}
	st := Status{
		Unit:        unit,
		Description: p["Description"],
		Load:        p["LoadState"],
		Active:      p["ActiveState"],
		Sub:         p["SubState"],
		Enabled:     p["UnitFileState"],
		Result:      p["Result"],
		File:        p["FragmentPath"],
		Memory:      -1,
	}
	st.MainPID, _ = strconv.Atoi(p["MainPID"])
	st.Restarts, _ = strconv.Atoi(p["NRestarts"])
	// MemoryCurrent is "[not set]" or 2^64-1 when not tracked.
	if m, err := strconv.ParseUint(p["MemoryCurrent"], 10, 64); err == nil && m < 1<<62 {
		st.Memory = int64(m)
	}
	mono, _ := strconv.ParseInt(p["ActiveEnterTimestampMonotonic"], 10, 64) // µs since boot
	up := strings.Fields(string(uptime))
	if st.Active == "active" && mono > 0 && len(up) > 0 {
		if upS, err := strconv.ParseFloat(up[0], 64); err == nil {
			st.ActiveFor = max(0, int64(upS)-mono/1_000_000)
		}
	}
	return st
}
