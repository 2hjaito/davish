package sysmon

import (
	"math"
	"os"
	"regexp"
	"strings"
	"testing"

	"davish/internal/platform"
)

// testdata/windows-procs.txt is real output of procs.ps1. windows-collect.txt
// and windows-info.txt are built by hand to match what collect.ps1 and
// info.ps1 are written to print: the scripts print the shapes of /proc so that
// the Linux parsers read them. Replace them with real output when available.

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

// testdata/windows-procs.txt is procs.ps1 on a real Windows machine (OpenSSH,
// cmd.exe as the SSH shell), cut to a dozen system processes.
func TestWindowsProcesses(t *testing.T) {
	snap, err := parseProcs(readSections(t, "windows-procs.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if snap.memTotal != 33376128*1024 || snap.clk != 1e7 || snap.page != 1 {
		t.Fatalf("snapshot = mem %d, clk %v, page %d", snap.memTotal, snap.clk, snap.page)
	}
	byCPU := snap.top(procSample{}, SortCPU, TopProcesses)
	if byCPU.Total != 12 {
		t.Fatalf("total = %d; want 12", byCPU.Total)
	}
	// The first call has no earlier sample, so CPU is the average over each
	// process's life: System has used 12967 of the 185614 seconds since it started.
	if top := byCPU.Procs[0]; top.PID != 4 || top.CPU != 7.0 || byCPU.Procs[1].PID != 3032 || byCPU.Procs[1].CPU != 3.3 {
		t.Errorf("busiest = %+v then %+v", top, byCPU.Procs[1])
	}

	byPID := map[int]Process{}
	for _, p := range byCPU.Procs {
		byPID[p.PID] = p
	}
	system := byPID[4]
	if system.Name != "System" || system.PPID != 0 || system.Threads != 246 || system.Mem != 24592384 {
		t.Errorf("System = %+v", system)
	}
	// Protected processes have no command line: shown in brackets like a kernel thread.
	if system.Command != "[System]" || byPID[2524].Command != "[Memory Compression]" {
		t.Errorf("commands = %q, %q", system.Command, byPID[2524].Command)
	}
	if math.Abs(system.Age-185613.71) > 0.01 {
		t.Errorf("System age = %v", system.Age)
	}
	// Windows has no uid table here; the user column shows the placeholder.
	if system.User != "-" {
		t.Errorf("user = %q", system.User)
	}

	sftp := byPID[4044]
	if want := `"c:\windows\system32\cmd.exe" /c "sftp-server.exe "`; sftp.Command != want || sftp.Name != "cmd.exe" || sftp.PPID != 2704 {
		t.Errorf("sftp shell = %+v", sftp)
	}
	if math.Abs(sftp.Age-2333.4) > 0.1 {
		t.Errorf("sftp age = %v", sftp.Age)
	}

	byMem := snap.top(procSample{}, SortMem, TopProcesses)
	if byMem.Procs[0].PID != 9128 || byMem.Procs[0].MemPct != 0.7 {
		t.Errorf("biggest = %+v", byMem.Procs[0])
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
