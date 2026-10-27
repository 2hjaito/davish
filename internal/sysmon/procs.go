package sysmon

import (
	"cmp"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"davish/internal/apperr"
)

//go:embed procs.sh
var procsScript string

// TopProcesses is how many processes a list holds; enough for a screen,
// small enough for a payload every few seconds.
const TopProcesses = 50

// maxCommand caps a command line; some (Java, Node) run to many kilobytes.
const maxCommand = 512

// Process is one row of the Monitor's process table.
type Process struct {
	PID     int     `json:"pid"`
	PPID    int     `json:"ppid"`
	User    string  `json:"user"`
	Name    string  `json:"name"`    // the kernel's short name (comm)
	Command string  `json:"command"` // full command line
	State   string  `json:"state"`   // R, S, D, Z…
	CPU     float64 `json:"cpu"`     // percent of one core, so it can pass 100 like in top
	Mem     uint64  `json:"mem"`     // resident bytes
	MemPct  float64 `json:"memPct"`
	Threads int     `json:"threads"`
	Age     float64 `json:"age"` // seconds since the process started
}

// ProcessList is the busiest processes and how many exist.
type ProcessList struct {
	Total int       `json:"total"`
	Procs []Process `json:"procs"`
}

// Sort keys for Processes.
const (
	SortCPU = "cpu"
	SortMem = "mem"
)

// procSample is what the next call needs to turn CPU time into load.
type procSample struct {
	uptime float64
	ticks  map[procKey]uint64
}

// procKey tells a reused PID apart from the process that had it before.
type procKey struct {
	pid   int
	start uint64
}

// Processes returns the TopProcesses busiest processes by sortBy. CPU load
// is measured since the previous call for the server; the first call falls
// back to each process's average over its lifetime.
func (s *Service) Processes(ctx context.Context, serverID, sortBy string) (ProcessList, error) {
	if sortBy != SortCPU && sortBy != SortMem {
		return ProcessList{}, apperr.New(apperr.InvalidInput, "unknown sort key")
	}
	client, err := s.clients.Client(serverID)
	if err != nil {
		return ProcessList{}, err
	}
	sections, err := runScript(ctx, client, procsScript)
	if err != nil {
		return ProcessList{}, apperr.Wrap(apperr.Internal, "could not read processes", err)
	}
	snap, err := parseProcs(sections)
	if err != nil {
		return ProcessList{}, apperr.Wrap(apperr.Internal, "could not read processes", err)
	}

	s.procsMu.Lock()
	prev := s.procs[serverID]
	s.procs[serverID] = snap.sample()
	s.procsMu.Unlock()

	return snap.top(prev, sortBy, TopProcesses), nil
}

// procSnapshot is one parsed run of procs.sh.
type procSnapshot struct {
	uptime   float64
	clk      float64 // ticks per second
	page     uint64  // bytes
	memTotal uint64  // bytes
	procs    []procStat
}

type procStat struct {
	Process
	ticks uint64 // user + system CPU time
	start uint64 // ticks after boot
}

func (p procSnapshot) sample() procSample {
	s := procSample{uptime: p.uptime, ticks: make(map[procKey]uint64, len(p.procs))}
	for _, ps := range p.procs {
		s.ticks[procKey{ps.PID, ps.start}] = ps.ticks
	}
	return s
}

// top fills in CPU and memory shares and keeps the n highest by sortBy.
func (p procSnapshot) top(prev procSample, sortBy string, n int) ProcessList {
	dt := p.uptime - prev.uptime
	out := make([]Process, 0, len(p.procs))
	for _, ps := range p.procs {
		pr := ps.Process
		pr.Age = max(0, p.uptime-float64(ps.start)/p.clk)
		if before, ok := prev.ticks[procKey{ps.PID, ps.start}]; ok && dt > 0 {
			pr.CPU = float64(delta(before, ps.ticks)) / p.clk / dt * 100
		} else if pr.Age > 0 {
			pr.CPU = float64(ps.ticks) / p.clk / pr.Age * 100
		}
		pr.CPU = round1(pr.CPU)
		if p.memTotal > 0 {
			pr.MemPct = round1(100 * float64(pr.Mem) / float64(p.memTotal))
		}
		out = append(out, pr)
	}
	slices.SortFunc(out, func(a, b Process) int {
		var c int
		if sortBy == SortMem {
			c = cmp.Compare(b.Mem, a.Mem)
		} else {
			c = cmp.Compare(b.CPU, a.CPU)
		}
		return cmp.Or(c, cmp.Compare(a.PID, b.PID))
	})
	return ProcessList{Total: len(out), Procs: out[:min(n, len(out))]}
}

func parseProcs(s map[string][]string) (procSnapshot, error) {
	var p procSnapshot
	var err error
	if p.uptime, err = firstFloat(s["uptime"], "uptime"); err != nil {
		return p, err
	}
	if p.clk, err = firstFloat(s["clk"], "clk"); err != nil || p.clk <= 0 {
		p.clk = 100 // the value on every mainstream Linux
	}
	page, err := firstFloat(s["pagesize"], "pagesize")
	if err != nil || page <= 0 {
		page = 4096
	}
	p.page = uint64(page)
	if mem, err := parseMeminfo(s["meminfo"]); err == nil {
		p.memTotal = mem.Total
	}

	users := parsePasswd(s["passwd"])
	type psRow struct {
		uid     string
		command string
	}
	rows := map[int]psRow{}
	for _, line := range s["ps"] {
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		pid, err := strconv.Atoi(f[0])
		if err != nil {
			continue
		}
		rows[pid] = psRow{uid: f[1], command: strings.Join(f[2:], " ")}
	}

	for _, line := range s["stat"] {
		ps, err := parseProcStat(line, p.page)
		if err != nil {
			continue // the process may have exited mid-read
		}
		if r, ok := rows[ps.PID]; ok {
			ps.User = cmp.Or(users[r.uid], r.uid)
			ps.Command = truncate(r.command, maxCommand)
		}
		if ps.Command == "" {
			ps.Command = "[" + ps.Name + "]" // a kernel thread, as ps shows it
		}
		p.procs = append(p.procs, ps)
	}
	if len(p.procs) == 0 {
		return p, errors.New("procs: no process read")
	}
	return p, nil
}

// parseProcStat reads /proc/<pid>/stat. The name sits in parentheses and
// may hold spaces or parentheses itself, so the fields after it are found
// from the last ')'.
func parseProcStat(line string, page uint64) (procStat, error) {
	open := strings.IndexByte(line, '(')
	end := strings.LastIndexByte(line, ')')
	if open < 0 || end < open {
		return procStat{}, fmt.Errorf("procs: bad stat line %q", line)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(line[:open]))
	if err != nil {
		return procStat{}, fmt.Errorf("procs: pid: %w", err)
	}
	// f[0] is field 3 of proc(5), so field n is f[n-3].
	f := strings.Fields(line[end+1:])
	if len(f) < 22 {
		return procStat{}, fmt.Errorf("procs: short stat line for %d", pid)
	}
	num := func(n int) uint64 {
		v, _ := strconv.ParseUint(f[n-3], 10, 64)
		return v
	}
	ppid, _ := strconv.Atoi(f[4-3])
	return procStat{
		Process: Process{
			PID:     pid,
			PPID:    ppid,
			Name:    line[open+1 : end],
			State:   f[0],
			Mem:     num(24) * page,
			Threads: int(num(20)), //nolint:gosec // a thread count fits an int
		},
		ticks: num(14) + num(15),
		start: num(22),
	}, nil
}

// parsePasswd maps UIDs to names from "name:uid" lines. Users from LDAP and
// the like are missing and show as their UID.
func parsePasswd(lines []string) map[string]string {
	users := map[string]string{}
	for _, line := range lines {
		name, uid, ok := strings.Cut(line, ":")
		if ok && name != "" {
			if _, dup := users[uid]; !dup {
				users[uid] = name
			}
		}
	}
	return users
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	// Cut on a rune boundary so the JSON stays valid UTF-8.
	for n > 0 && !isRuneStart(s[n]) {
		n--
	}
	return s[:n] + "…"
}

func isRuneStart(b byte) bool { return b&0xC0 != 0x80 }
