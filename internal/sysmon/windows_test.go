package sysmon

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"davish/internal/platform"
)

// testdata/windows-*.txt are what collect.ps1, procs.ps1 and info.ps1 are
// written to print, built by hand: the scripts print the shapes of /proc so
// that the Linux parsers read them. They are checked against a real server
// by running the scripts there (see the pull request).

func TestWindowsCollectorReadsLikeLinux(t *testing.T) {
	f, err := os.Open("testdata/windows-collect.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	r := NewReader(f)
	first, err := r.Next()
	if err != nil {
		t.Fatal(err)
	}
	second, err := r.Next()
	if err != nil {
		t.Fatal(err)
	}

	if first.Health == nil || len(first.Health.Failed) != 1 || first.Health.Failed[0] != "Spooler.service" {
		t.Errorf("failed services = %+v", first.Health)
	}
	if got := first.Health.Containers; len(got) != 2 || got[1] != (ContainerState{Name: "db", Restarts: 7, Status: "restarting"}) {
		t.Errorf("containers = %+v", got)
	}
	if second.Health != nil {
		t.Errorf("health on a snapshot that skips it: %+v", second.Health)
	}

	tick := Rate("win", first, second)
	// Two cores, each 2s of wall clock: user 0.3s, kernel 0.1s, idle 1.6s.
	if tick.CPU != 20 || tick.CPUSplit.User != 15 || tick.CPUSplit.System != 5 {
		t.Errorf("cpu = %v %+v; want 20 {User:15 System:5}", tick.CPU, tick.CPUSplit)
	}
	if len(tick.Cores) != 2 || tick.Cores[0] != 20 || tick.Cores[1] != 20 {
		t.Errorf("cores = %v", tick.Cores)
	}
	if tick.MemTotal != 8<<30 || tick.MemUsed != (8388608-4000000)*1024 {
		t.Errorf("memory = %d of %d", tick.MemUsed, tick.MemTotal)
	}
	if tick.SwapTotal != 2<<30 || tick.SwapUsed != 524288*1024 {
		t.Errorf("swap = %d of %d", tick.SwapUsed, tick.SwapTotal)
	}
	if tick.Procs != 88 || tick.Uptime != 1002 {
		t.Errorf("procs = %d, uptime = %v", tick.Procs, tick.Uptime)
	}

	// The system drive is the "/": first, and the figure the menu bar shows.
	if len(tick.Mounts) != 2 || tick.Mounts[0].Path != `C:\` || tick.Mounts[1].Path != `D:\` {
		t.Fatalf("mounts = %+v", tick.Mounts)
	}
	if tick.DiskPct != 50 || tick.Mounts[0].Size != 100<<30 {
		t.Errorf("C: = %v%% of %d", tick.DiskPct, tick.Mounts[0].Size)
	}

	// An interface name may hold spaces; idle tunnel adapters are left out.
	if len(tick.Ifaces) != 1 || tick.Ifaces[0].Name != "Intel[R] 82574L Gigabit Network Connection" {
		t.Fatalf("ifaces = %+v", tick.Ifaces)
	}
	if tick.RxBps != 100000 || tick.TxBps != 50000 {
		t.Errorf("network = %v rx, %v tx bytes/s", tick.RxBps, tick.TxBps)
	}
	if len(tick.Disks) != 1 || tick.ReadBps != 8000*512/2 {
		t.Errorf("disks = %+v read %v", tick.Disks, tick.ReadBps)
	}
	if tick.WriteBps != 4000*512/2 {
		t.Errorf("write = %v", tick.WriteBps)
	}
}

func TestWindowsProcesses(t *testing.T) {
	snap, err := parseProcs(readSections(t, "windows-procs.txt"))
	if err != nil {
		t.Fatal(err)
	}
	list := snap.top(procSample{}, SortMem, TopProcesses)
	if list.Total != 3 {
		t.Fatalf("total = %d", list.Total)
	}
	nginx := list.Procs[0]
	if nginx.PID != 2210 || nginx.Name != "nginx.exe" || nginx.Mem != 64<<20 || nginx.Threads != 4 || nginx.PPID != 1 {
		t.Errorf("nginx = %+v", nginx)
	}
	if want := `"C:\Program Files\nginx\nginx.exe" -c conf\nginx.conf`; nginx.Command != want {
		t.Errorf("command = %q", nginx.Command)
	}
	// Started 900s after boot, 102s ago; 10s of CPU over that life.
	if nginx.Age != 102 || nginx.CPU != 9.8 {
		t.Errorf("age %v, cpu %v", nginx.Age, nginx.CPU)
	}
	// The System process has no command line: shown in brackets like a kernel thread.
	var system Process
	for _, p := range list.Procs {
		if p.PID == 4 {
			system = p
		}
	}
	if system.Command != "[System]" {
		t.Errorf("System command = %q", system.Command)
	}
}

func TestWindowsInfo(t *testing.T) {
	info := parseInfo(readSections(t, "windows-info.txt"))
	want := SystemInfo{
		Hostname: "WIN-SRV01",
		OS:       "Windows Server 2022 Datacenter",
		Kernel:   "10.0.20348",
		Arch:     "x86_64",
		CPUModel: "Intel(R) Xeon(R) Platinum 8272CL CPU @ 2.60GHz",
		Virt:     "microsoft",
	}
	if info != want {
		t.Errorf("info = %+v; want %+v", info, want)
	}
}

// PowerShell reads "$name:" as a variable in a drive-like scope and refuses
// to parse the script. Only the real scopes may be followed by a colon.
func TestWindowsScriptsHaveNoScopeLikeVariables(t *testing.T) {
	tg, _ := targetFor(platform.Windows)
	bad := regexp.MustCompile(`\$([A-Za-z_][A-Za-z0-9_]*):`)
	scopes := map[string]bool{"env": true, "script": true, "global": true, "local": true, "private": true}
	for name, s := range map[string]string{"collect": tg.collect, "procs": tg.procs, "info": tg.info} {
		for _, m := range bad.FindAllStringSubmatch(s, -1) {
			if !scopes[strings.ToLower(m[1])] {
				t.Errorf("%s.ps1: %s: is read as a scope; write ${%s}:", name, m[0], m[1])
			}
		}
	}
}

func TestWindowsTarget(t *testing.T) {
	tg, ok := targetFor(platform.Windows)
	if !ok {
		t.Fatal("Windows must be supported")
	}
	for name, s := range map[string]string{"collect": tg.collect, "procs": tg.procs, "info": tg.info} {
		if !strings.Contains(s, "@@end") {
			t.Errorf("%s.ps1 never prints @@end", name)
		}
		if strings.Count(s, "{") != strings.Count(s, "}") {
			t.Errorf("%s.ps1 has unbalanced braces", name)
		}
		if strings.Contains(s, "\r") {
			t.Errorf("%s.ps1 has CR line endings; the script travels as LF", name)
		}
	}
	// Go sets the pace in PowerShell syntax, not sh's.
	if got := tg.vars(2, 15); got != "$every = 2\n$healthEvery = 15\n" {
		t.Errorf("vars = %q", got)
	}
	cmd, stdin := tg.launch("Write-Output 1")
	if !strings.HasPrefix(cmd, "powershell.exe ") || stdin != "Write-Output 1" {
		t.Errorf("launch = %q, %q", cmd, stdin)
	}
	// Both scripts use the section names the parser reads.
	for _, sec := range []string{"@@stat", "@@meminfo", "@@loadavg", "@@uptime", "@@netdev", "@@df", "@@diskstats", "@@block", "@@gpu", "@@failed", "@@containers"} {
		if !strings.Contains(tg.collect, sec) {
			t.Errorf("collect.ps1 lacks %s", sec)
		}
	}
}
