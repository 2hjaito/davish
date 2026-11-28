package sysmon

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

var testCards = []drmCard{
	{slot: "0000:03:00.0", index: 1, total: 8 << 30},
	{slot: "0000:00:02.0", index: 2, total: -1},
}

func TestParseDRMFD(t *testing.T) {
	lines := []string{
		// pid 100: one amdgpu client held by two descriptors (a dup), counted once.
		"/proc/100/fdinfo/7:drm-driver:\tamdgpu",
		"/proc/100/fdinfo/7:drm-pdev:\t0000:03:00.0",
		"/proc/100/fdinfo/7:drm-client-id:\t42",
		"/proc/100/fdinfo/7:drm-engine-gfx:\t5000000000 ns",
		"/proc/100/fdinfo/7:drm-engine-compute:\t1000000000 ns",
		"/proc/100/fdinfo/7:drm-memory-vram:\t2048 KiB",
		"/proc/100/fdinfo/9:drm-driver:\tamdgpu",
		"/proc/100/fdinfo/9:drm-pdev:\t0000:03:00.0",
		"/proc/100/fdinfo/9:drm-client-id:\t42",
		"/proc/100/fdinfo/9:drm-engine-gfx:\t5000000000 ns",
		"/proc/100/fdinfo/9:drm-memory-vram:\t2048 KiB",
		// a second client of the same process adds up
		"/proc/100/fdinfo/11:drm-driver:\tamdgpu",
		"/proc/100/fdinfo/11:drm-pdev:\t0000:03:00.0",
		"/proc/100/fdinfo/11:drm-client-id:\t43",
		"/proc/100/fdinfo/11:drm-engine-gfx:\t1000000000 ns",
		"/proc/100/fdinfo/11:drm-memory-vram:\t1024 KiB",
		// pid 200 on the Intel card: i915 names its render engine "render", memory in system0 only
		"/proc/200/fdinfo/4:drm-driver:\ti915",
		"/proc/200/fdinfo/4:drm-pdev:\t0000:00:02.0",
		"/proc/200/fdinfo/4:drm-client-id:\t7",
		"/proc/200/fdinfo/4:drm-engine-render:\t300 ns",
		// a descriptor of an unknown card, and one with no PID in its path
		"/proc/300/fdinfo/4:drm-driver:\tnouveau",
		"/proc/300/fdinfo/4:drm-pdev:\t0000:99:00.0",
		"garbage",
	}
	got := parseDRMFD(lines, testCards)
	if len(got) != 2 {
		t.Fatalf("got %d rows: %+v", len(got), got)
	}
	amd := got[engineKey{gpu: 1, pid: 100}]
	if amd == nil || amd.mem != 3072*1024 || amd.engines["gfx"] != 6e9 || amd.engines["compute"] != 1e9 || !amd.render {
		t.Errorf("amd = %+v", amd)
	}
	intel := got[engineKey{gpu: 2, pid: 200}]
	if intel == nil || intel.mem != -1 || intel.engines["render"] != 300 || !intel.render {
		t.Errorf("intel = %+v", intel)
	}
}

func TestParseDRMFDWithoutPdevUsesTheOnlyCard(t *testing.T) {
	lines := []string{"/proc/5/fdinfo/3:drm-driver:\tamdgpu", "/proc/5/fdinfo/3:drm-engine-compute:\t10 ns"}
	if got := parseDRMFD(lines, testCards[:1]); len(got) != 1 || got[engineKey{gpu: 1, pid: 5}].render {
		t.Errorf("got %+v", got)
	}
	if got := parseDRMFD(lines, testCards); len(got) != 0 {
		t.Errorf("with two cards a descriptor without a slot is not guessed: %+v", got)
	}
}

func TestEngineLoad(t *testing.T) {
	cur := map[string]float64{"gfx": 7e9, "compute": 3.5e9}
	prev := map[string]float64{"gfx": 6e9, "compute": 3e9}
	if got := engineLoad(cur, prev, 2); got != 50 {
		t.Errorf("busiest engine used 1 s of 2: %v", got)
	}
	if engineLoad(cur, nil, 2) != -1 || engineLoad(cur, prev, 0) != -1 {
		t.Error("nothing to compare is -1")
	}
	if got := engineLoad(map[string]float64{"gfx": 9e9}, map[string]float64{"gfx": 1e9}, 1); got != 100 {
		t.Errorf("several queues can add up past 100: %v", got)
	}
	if got := engineLoad(map[string]float64{"gfx": 1}, map[string]float64{"gfx": 9}, 1); got != 0 {
		t.Errorf("a counter that went back (a new client) is 0: %v", got)
	}
}

func TestGPUProcsAddsAMDAndIntelRows(t *testing.T) {
	s := readSections(t, "gpuprocs-idle.txt")
	s["drmcards"] = []string{"0000:03:00.0|0x1002|8589934592"}
	s["drmfd"] = []string{
		"/proc/100/fdinfo/7:drm-driver:\tamdgpu",
		"/proc/100/fdinfo/7:drm-pdev:\t0000:03:00.0",
		"/proc/100/fdinfo/7:drm-client-id:\t1",
		"/proc/100/fdinfo/7:drm-engine-gfx:\t4000000000 ns",
		"/proc/100/fdinfo/7:drm-memory-vram:\t1048576 KiB",
	}
	s["uptime"] = []string{"1000.00 900.00"}
	snap, err := parseGPUProcs(s)
	if err != nil {
		t.Fatal(err)
	}
	first := snap.withEngineLoad(snap.list(procSample{}), engineSample{})
	next := 1 // the fixture has one NVIDIA GPU, index 0
	r := gpuRow(t, first, next, 100)
	if r.Type != "G" || r.GPUMem != 1<<30 || r.GPUMemPct != 12.5 || r.GPUUtil != -1 {
		t.Errorf("first reading = %+v", r)
	}
	// Two seconds later the client has used 1 more second of gfx time.
	prev := snap.engines()
	s["drmfd"][3] = "/proc/100/fdinfo/7:drm-engine-gfx:\t5000000000 ns"
	s["uptime"] = []string{"1002.00 900.00"}
	snap2, err := parseGPUProcs(s)
	if err != nil {
		t.Fatal(err)
	}
	r = gpuRow(t, snap2.withEngineLoad(snap2.list(procSample{}), prev), next, 100)
	if r.GPUUtil != 50 {
		t.Errorf("load = %v, want 50", r.GPUUtil)
	}
}

// The real script, against fake /sys/class/drm and a fake /proc fdinfo tree.
func TestGPUProcsScriptReadsDRMFDInfo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("gpuprocs.sh needs sh")
	}
	pid := os.Getpid()
	proc := t.TempDir()
	fdinfo := filepath.Join(proc, strconv.Itoa(pid), "fdinfo")
	if err := os.MkdirAll(fdinfo, 0o750); err != nil {
		t.Fatal(err)
	}
	body := "pos:\t0\nflags:\t02\ndrm-driver:\tamdgpu\ndrm-pdev:\t0000:03:00.0\ndrm-client-id:\t42\n" +
		"drm-engine-gfx:\t5000000000 ns\ndrm-memory-vram:\t2048 KiB\ndrm-memory-gtt:\t99 KiB\n"
	for name, text := range map[string]string{"7": body, "8": "pos:\t0\nflags:\t02\n"} {
		if err := os.WriteFile(filepath.Join(fdinfo, name), []byte(text), 0o644); err != nil { //nolint:gosec // fake procfs
			t.Fatal(err)
		}
	}
	// With no nvidia-smi on the PATH, the NVIDIA cards would be listed too; the fake tree has none.
	cmd := exec.Command("sh", "-s")
	cmd.Stdin = strings.NewReader(gpuProcsScript)
	cmd.Env = append(os.Environ(), "PATH=/usr/bin:/bin", "DAVISH_DRM="+fakeDRM(t), "DAVISH_PROC="+proc)
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewReader(bytes.NewReader(out)).sections()
	if err != nil {
		t.Fatalf("%v in:\n%s", err, out)
	}
	if len(s["drmcards"]) != 2 {
		t.Errorf("cards = %q", s["drmcards"])
	}
	snap, err := parseGPUProcs(s)
	if err != nil {
		t.Fatal(err)
	}
	list := snap.list(procSample{})
	if len(list.Procs) != 1 {
		t.Fatalf("rows = %+v from %q", list.Procs, s["drmfd"])
	}
	r := list.Procs[0]
	if r.PID != pid || r.GPU != 0 || r.GPUMem != 2048*1024 || r.Type != "G" {
		t.Errorf("row = %+v", r)
	}
}

// What gpuprocs.ps1 prints for an AMD card on Windows: the card named by "*"
// (the only adapter), engine times in ns and memory in bytes.
func TestParseDRMFDFromWindows(t *testing.T) {
	cards := parseDRMCards([]string{"*|0x1002|8589934592"}, 0)
	lines := []string{
		"/proc/4812/fdinfo/1:drm-driver:\twddm",
		"/proc/4812/fdinfo/1:drm-pdev:\t*",
		"/proc/4812/fdinfo/1:drm-engine-3d:\t900000000 ns",
		"/proc/4812/fdinfo/1:drm-engine-copy:\t5000 ns",
		"/proc/4812/fdinfo/1:drm-memory-vram:\t268435456 B",
	}
	got := parseDRMFD(lines, cards)[engineKey{gpu: 0, pid: 4812}]
	if got == nil || got.mem != 256*1024*1024 || got.engines["3d"] != 9e8 || !got.render {
		t.Errorf("got %+v", got)
	}
}
