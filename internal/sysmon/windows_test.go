package sysmon

import (
	"math"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"davish/internal/platform"
)

// testdata/windows-procs.txt, windows-collect-real.txt and windows-info.txt are
// real output of the scripts (the host name replaced). windows-collect.txt is
// built by hand to cover what that machine lacked: a failed service,
// containers and a second drive. The scripts print the shapes of /proc so that
// the Linux parsers read them.

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

// testdata/windows-collect-real.txt is collect.ps1 on a real Windows machine
// (six cores, three network adapters, two disks), as the first version of the
// script printed it: the CPU "system" column held the 1601-based wall clock, so
// the counters are 18 digits long. Only differences are used, so it still reads.
// The expected figures were worked out from the raw numbers separately.
func TestWindowsCollectorOnRealOutput(t *testing.T) {
	f, err := os.Open("testdata/windows-collect-real.txt")
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
	tick := Rate("win", first, second)

	if tick.CPU != 38.7 || tick.CPUSplit.User != 8.2 || tick.CPUSplit.System != 30.4 {
		t.Errorf("cpu = %v %+v; want 38.7 {User:8.2 System:30.4}", tick.CPU, tick.CPUSplit)
	}
	if want := []float64{30.9, 27.0, 60.3, 59.2, 30.9, 23.9}; !slices.Equal(tick.Cores, want) {
		t.Errorf("cores = %v; want %v", tick.Cores, want)
	}
	if tick.MemTotal != 33376128*1024 || tick.MemUsed != 8058912768 || tick.SwapTotal != 2<<30 || tick.SwapUsed != 0 {
		t.Errorf("memory = %d of %d, swap %d of %d", tick.MemUsed, tick.MemTotal, tick.SwapUsed, tick.SwapTotal)
	}
	if tick.Uptime != 185780.67 || tick.Procs != 187 {
		t.Errorf("uptime %v, procs %d", tick.Uptime, tick.Procs)
	}
	if len(tick.Mounts) != 1 || tick.Mounts[0].Path != `C:\` || tick.DiskPct != 8 {
		t.Errorf("mounts = %+v, pct %v", tick.Mounts, tick.DiskPct)
	}
	// The idle wireless adapter is left out; the three busy ones are summed.
	if len(tick.Ifaces) != 3 || tick.Ifaces[1].Name != "ASIX AX88179 USB 3.0 to Gigabit Ethernet Adapter" {
		t.Errorf("ifaces = %+v", tick.Ifaces)
	}
	if math.Abs(tick.RxBps-3463029.5) > 0.1 || math.Abs(tick.TxBps-6775650) > 0.1 {
		t.Errorf("network = %v rx, %v tx bytes/s", tick.RxBps, tick.TxBps)
	}
	if len(tick.Disks) != 2 || math.Abs(tick.ReadBps-23272.7) > 0.1 || math.Abs(tick.WriteBps-498036.4) > 0.1 {
		t.Errorf("disks = %+v, read %v, write %v", tick.Disks, tick.ReadBps, tick.WriteBps)
	}
	// No failed service, no container and no GPU on this machine.
	if first.Health == nil || len(first.Health.Failed) != 0 || len(first.Health.Containers) != 0 || len(tick.GPUs) != 0 {
		t.Errorf("health = %+v, gpus = %v", first.Health, tick.GPUs)
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
		Hostname: "WIN-HOST",
		OS:       "Windows 11 Pro",
		Kernel:   "10.0.26200",
		Arch:     "x86_64",
		CPUModel: "Intel(R) Core(TM) i5-9400 CPU @ 2.90GHz",
		Virt:     "", // "none": bare metal
	}
	if info != want {
		t.Errorf("info = %+v; want %+v", info, want)
	}
}

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
