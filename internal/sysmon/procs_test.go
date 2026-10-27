package sysmon

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func readSections(t *testing.T, name string) map[string][]string {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", filepath.Clean(name)))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	s, err := NewReader(f).sections()
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// testdata/procs.txt is procs.sh on the test sshd, seen from a session: the
// ps that listed processes exited before /proc was read.
func TestParseProcsRealOutput(t *testing.T) {
	snap, err := parseProcs(readSections(t, "procs.txt"))
	if err != nil {
		t.Fatal(err)
	}
	list := snap.top(procSample{}, SortMem, TopProcesses)

	if list.Total != 4 {
		t.Fatalf("total = %d; want 4", list.Total)
	}
	var pids []int
	for _, p := range list.Procs {
		pids = append(pids, p.PID)
	}
	if want := []int{18788, 1, 18799, 18800}; !slices.Equal(pids, want) {
		t.Errorf("by memory: pids %v; want %v", pids, want)
	}

	init := list.Procs[1]
	want := Process{
		PID: 1, User: "root", Name: "sshd", State: "S", Threads: 1,
		Command: "sshd: /usr/sbin/sshd -D -e [listener] 0 of 10-100 startups",
		Mem:     1749 * 4096,
		MemPct:  0.1,
		// 2 ticks of CPU over a life of 5998.18 - 15.29 seconds.
		CPU: 0,
		Age: 5998.18 - 15.29,
	}
	if init.Age-want.Age > 1e-6 || want.Age-init.Age > 1e-6 {
		t.Errorf("age = %v; want %v", init.Age, want.Age)
	}
	init.Age = want.Age
	if init != want {
		t.Errorf("pid 1:\n got %+v\nwant %+v", init, want)
	}
	if sh := list.Procs[3]; sh.User != "tester" || sh.Command != "sh -s" || sh.PPID != 18799 {
		t.Errorf("sh = %+v", sh)
	}
}

func TestTopMeasuresLoadSinceTheLastCall(t *testing.T) {
	proc := func(pid int, ticks, start uint64) procStat {
		return procStat{Process: Process{PID: pid}, ticks: ticks, start: start}
	}
	prev := procSnapshot{uptime: 100, clk: 100, procs: []procStat{
		proc(1, 1000, 0),
		proc(2, 500, 1000),
		proc(3, 900, 2000), // gone by the next call; its PID comes back as a new process
	}}.sample()
	cur := procSnapshot{uptime: 102, clk: 100, procs: []procStat{
		proc(1, 1000+100, 0),   // half a core over 2s
		proc(2, 500+400, 1000), // two cores
		proc(3, 60, 9800),      // started at 98s, 0.6s of CPU since: lifetime average
	}}
	list := cur.top(prev, SortCPU, 2)

	if list.Total != 3 || len(list.Procs) != 2 {
		t.Fatalf("total %d, %d listed; want 3, 2", list.Total, len(list.Procs))
	}
	if p := list.Procs[0]; p.PID != 2 || p.CPU != 200 {
		t.Errorf("first = pid %d at %v%%; want pid 2 at 200%%", p.PID, p.CPU)
	}
	if p := list.Procs[1]; p.PID != 1 || p.CPU != 50 {
		t.Errorf("second = pid %d at %v%%; want pid 1 at 50%%", p.PID, p.CPU)
	}
	all := cur.top(prev, SortCPU, 10)
	if p := all.Procs[2]; p.PID != 3 || p.CPU != 15 {
		t.Errorf("new process = pid %d at %v%%; want pid 3 at 15%%", p.PID, p.CPU)
	}
}

func TestParseProcStatOddNames(t *testing.T) {
	rest := " S 1 2 3 0 -1 0 0 0 0 0 7 3 0 0 20 0 4 0 500 0 10"
	tests := []struct{ line, name string }{
		{"42 (sd-pam)" + rest, "sd-pam"},
		{"42 (tmux: server)" + rest, "tmux: server"},
		{"42 (a) b (c))" + rest, "a) b (c)"},
	}
	for _, tt := range tests {
		p, err := parseProcStat(tt.line, 4096)
		if err != nil {
			t.Fatalf("%q: %v", tt.line, err)
		}
		if p.PID != 42 || p.Name != tt.name || p.ticks != 10 || p.Threads != 4 || p.start != 500 || p.Mem != 10*4096 {
			t.Errorf("%q: got %+v", tt.line, p)
		}
	}
	if _, err := parseProcStat("42 (short) S 1 2", 4096); err == nil {
		t.Error("short line parsed")
	}
}

func TestTruncateKeepsUTF8(t *testing.T) {
	s := strings.Repeat("a", 9) + "é" // é is two bytes, at 9 and 10
	if got := truncate(s, 10); got != strings.Repeat("a", 9)+"…" {
		t.Errorf("got %q", got)
	}
	if got := truncate("short", 10); got != "short" {
		t.Errorf("got %q", got)
	}
}

func TestParseInfoRealOutput(t *testing.T) {
	got := parseInfo(readSections(t, "info.txt"))
	// lscpu prints "-" for the model inside this ARM VM.
	want := SystemInfo{
		Hostname: "3b58e3ca0393",
		OS:       "Ubuntu 24.04.5 LTS",
		Kernel:   "6.12.76-linuxkit",
		Arch:     "aarch64",
	}
	if got != want {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}

func TestParseInfoFallbacks(t *testing.T) {
	got := parseInfo(map[string][]string{
		"os":   {`NAME="CentOS Linux"`, `VERSION="7 (Core)"`},
		"cpu":  {" Intel(R) Xeon(R) CPU E5-2680 v4 @ 2.40GHz"},
		"virt": {"kvm"},
	})
	if got.OS != "CentOS Linux 7 (Core)" || got.CPUModel != "Intel(R) Xeon(R) CPU E5-2680 v4 @ 2.40GHz" || got.Virt != "kvm" {
		t.Errorf("got %+v", got)
	}
}
