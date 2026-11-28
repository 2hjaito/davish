package sysmon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"davish/internal/platform"
)

// fakeNvidiaSMI answers the three queries gpuprocs.sh makes: one compute
// process ($APP_PID) and, in pmon, a graphics one ($GFX_PID), the compute
// one again and an idle GPU's placeholder.
const fakeNvidiaSMI = `#!/bin/sh
case "$*" in
*query-gpu*) echo '0, GPU-test, 1024' ;;
*query-compute-apps*) echo "GPU-test, $APP_PID, 100, /usr/bin/fake" ;;
pmon*)
	echo '# gpu        pid  type    sm   mem   enc   dec    fb   command'
	echo '# Idx          #   C/G     %     %     %     %    MB   name'
	echo "    0   $GFX_PID     G     1     0     -     -    10   Xorg"
	echo "    0   $APP_PID     C     5     1     -     -   100   fake"
	echo '    1          -     -     -     -     -     -     -   -'
	;;
esac
`

// The script asks ps and /proc about the GPU PIDs only, each once.
func TestGPUProcsScriptReadsOnlyGPUPIDs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("gpuprocs.sh needs sh")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "nvidia-smi"), []byte(fakeNvidiaSMI), 0o755); err != nil { //nolint:gosec // an executable test double
		t.Fatal(err)
	}
	app, gfx := os.Getpid(), os.Getppid()
	cmd := exec.Command("sh", "-s")
	cmd.Stdin = strings.NewReader(gpuProcsScript)
	cmd.Env = append(os.Environ(),
		"PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"APP_PID="+strconv.Itoa(app), "GFX_PID="+strconv.Itoa(gfx))
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewReader(bytes.NewReader(out)).sections()
	if err != nil {
		t.Fatalf("%v in:\n%s", err, out)
	}
	if !slices.Equal(s["gpus"], []string{"0, GPU-test, 1024"}) || len(s["apps"]) != 1 || len(s["pmon"]) != 5 {
		t.Errorf("nvidia-smi sections = %q, %q, %q", s["gpus"], s["apps"], s["pmon"])
	}
	var pids []int
	for _, line := range s["ps"] {
		if f := strings.Fields(line); len(f) > 0 {
			pid, _ := strconv.Atoi(f[0])
			pids = append(pids, pid)
		}
	}
	slices.Sort(pids)
	want := []int{min(app, gfx), max(app, gfx)}
	if !slices.Equal(pids, want) {
		t.Errorf("ps listed %v; want only the GPU PIDs %v", pids, want)
	}
}

// Without nvidia-smi the script still ends, with nothing to list.
func TestGPUProcsScriptWithoutNvidiaSMI(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("gpuprocs.sh needs sh")
	}
	for _, dir := range []string{"/usr/bin", "/bin"} {
		if _, err := os.Stat(filepath.Join(dir, "nvidia-smi")); err == nil {
			t.Skip("this machine has nvidia-smi")
		}
	}
	cmd := exec.Command("sh", "-s")
	cmd.Stdin = strings.NewReader(gpuProcsScript)
	// A machine with an AMD or Intel card would list its processes: look at an empty tree.
	cmd.Env = append(os.Environ(), "PATH=/usr/bin:/bin", "DAVISH_DRM="+t.TempDir())
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewReader(bytes.NewReader(out)).sections()
	if err != nil {
		t.Fatalf("%v in:\n%s", err, out)
	}
	if len(s["apps"])+len(s["pmon"])+len(s["ps"])+len(s["stat"]) != 0 {
		t.Errorf("sections = %q", s)
	}
}

const mib = 1024 * 1024

// testdata/gpuprocs*.txt are built by hand after nvtop on a workstation with
// an RTX PRO 6000 (llama-server, an API server and a Whisper job), with a
// second GPU and an Xorg graphics process added.

func gpuSnap(t *testing.T, name string) gpuProcSnapshot {
	t.Helper()
	snap, err := parseGPUProcs(readSections(t, name))
	if err != nil {
		t.Fatal(err)
	}
	return snap
}

func gpuRow(t *testing.T, list GPUProcessList, gpu, pid int) GPUProcess {
	t.Helper()
	for _, p := range list.Procs {
		if p.GPU == gpu && p.PID == pid {
			return p
		}
	}
	t.Fatalf("no row for GPU %d PID %d in %+v", gpu, pid, list.Procs)
	return GPUProcess{}
}

func TestGPUProcsMergesAppsAndPmon(t *testing.T) {
	list := gpuSnap(t, "gpuprocs.txt").list(procSample{})

	if len(list.Procs) != 5 {
		t.Fatalf("rows = %d; want 5: %+v", len(list.Procs), list.Procs)
	}
	// Largest GPU memory first, wherever it sits.
	var keys [][2]int
	for _, p := range list.Procs {
		keys = append(keys, [2]int{p.GPU, p.PID})
	}
	want := [][2]int{{0, 3324900}, {0, 2869697}, {1, 3324900}, {0, 3157412}, {0, 4242}}
	if !slices.Equal(keys, want) {
		t.Errorf("order = %v; want %v", keys, want)
	}

	llama := list.Procs[0]
	if llama.Type != "C" || llama.GPUUtil != 12 || llama.GPUMem != 21382*mib || llama.GPUMemPct != 21.8 {
		t.Errorf("llama GPU side = %+v", llama)
	}
	if llama.User != "root" || llama.Name != "llama-server" || llama.Mem != 382208*4096 {
		t.Errorf("llama host side = %+v", llama)
	}
	if want := "/usr/lib/ollama/llama-server --model /root/.ollama/models/blobs/sha256-6a0746a1 --port 41117"; llama.Command != want {
		t.Errorf("command = %q", llama.Command)
	}
	if llama.CPU < 0 {
		t.Errorf("llama CPU = %v; want a reading", llama.CPU)
	}

	// pmon had no sample for it in that second.
	if api := gpuRow(t, list, 0, 2869697); api.GPUUtil != -1 {
		t.Errorf("api server util = %v; want -1", api.GPUUtil)
	}

	// A graphics process: only pmon knows it, and /proc does not show it.
	xorg := gpuRow(t, list, 0, 4242)
	if xorg.Type != "G" || xorg.GPUMem != 45*mib || xorg.GPUUtil != 1 {
		t.Errorf("Xorg GPU side = %+v", xorg)
	}
	if xorg.User != "" || xorg.Name != "Xorg" || xorg.Command != "Xorg" || xorg.CPU != -1 || xorg.Mem != -1 {
		t.Errorf("Xorg host side = %+v", xorg)
	}
}

func TestGPUProcsSameProcessOnTwoGPUs(t *testing.T) {
	list := gpuSnap(t, "gpuprocs.txt").list(procSample{})
	on1 := gpuRow(t, list, 1, 3324900)
	if on1.GPUMem != 1024*mib || on1.GPUUtil != 5 || on1.User != "root" {
		t.Errorf("llama on GPU 1 = %+v", on1)
	}
	gpuRow(t, list, 0, 3324900)
}

func TestGPUProcsOlderPmon(t *testing.T) {
	list := gpuSnap(t, "gpuprocs-old.txt").list(procSample{})
	if len(list.Procs) != 1 {
		t.Fatalf("rows = %+v; want one, the idle GPU's placeholder skipped", list.Procs)
	}
	p := list.Procs[0]
	// compute-apps said [N/A]; pmon's fb column fills in.
	if p.GPU != 0 || p.PID != 5150 || p.Type != "C" || p.GPUUtil != 40 || p.GPUMem != 300*mib || p.GPUMemPct != 1.2 {
		t.Errorf("row = %+v", p)
	}
	if p.User != "ml" || p.Command != "python3 train.py --epochs 10" {
		t.Errorf("host side = %+v", p)
	}
}

func TestGPUProcsWithoutPmon(t *testing.T) {
	list := gpuSnap(t, "gpuprocs-nopmon.txt").list(procSample{})
	if len(list.Procs) != 4 {
		t.Fatalf("rows = %d; want the 4 compute rows: %+v", len(list.Procs), list.Procs)
	}
	for _, p := range list.Procs {
		if p.Type != "C" || p.GPUUtil != -1 || p.GPUMem <= 0 {
			t.Errorf("row = %+v; want compute, util unknown, memory known", p)
		}
	}
}

func TestGPUProcsIdle(t *testing.T) {
	list := gpuSnap(t, "gpuprocs-idle.txt").list(procSample{})
	if list.Procs == nil || len(list.Procs) != 0 {
		t.Fatalf("procs = %#v; want an empty, non-nil slice", list.Procs)
	}
	b, err := json.Marshal(list)
	if err != nil || string(b) != `{"procs":[]}` {
		t.Errorf("json = %s, %v", b, err)
	}
}

func TestGPUProcsWithoutGPUList(t *testing.T) {
	s := readSections(t, "gpuprocs.txt")
	s["gpus"] = nil
	if _, err := parseGPUProcs(s); err == nil {
		t.Error("compute rows without a GPU list were accepted")
	}
}

func TestWindowsGPUProcs(t *testing.T) {
	list := gpuSnap(t, "windows-gpuprocs.txt").list(procSample{})
	if len(list.Procs) != 1 {
		t.Fatalf("rows = %+v", list.Procs)
	}
	p := list.Procs[0]
	// WDDM: the driver does not tell memory per process.
	if p.GPUMem != -1 || p.GPUMemPct != -1 || p.GPUUtil != -1 {
		t.Errorf("GPU side = %+v; want unknowns", p)
	}
	if p.User != "-" || p.Name != "infer.exe" || p.Command != `C:\app\infer.exe --serve` || p.Mem != 2147483648 {
		t.Errorf("host side = %+v", p)
	}
}

func TestGPUProcsCPUSinceLastCall(t *testing.T) {
	snap := gpuSnap(t, "gpuprocs.txt")
	prev := snap.sample()
	key := procKey{pid: 3324900, start: 100000}
	if _, ok := prev.ticks[key]; !ok {
		t.Fatalf("sample has no llama-server: %+v", prev)
	}
	// One second of CPU time in the last second: one full core.
	prev.ticks[key] -= 100
	prev.uptime--
	list := snap.list(prev)
	for _, gpu := range []int{0, 1} {
		if got := gpuRow(t, list, gpu, 3324900).CPU; got != 100 {
			t.Errorf("GPU %d llama CPU = %v; want 100", gpu, got)
		}
	}
}

// The Windows script cannot run here; it must at least print what the
// parser reads from the Linux one, in the same order.
func TestGPUProcsScriptsPrintTheSameSections(t *testing.T) {
	sections := regexp.MustCompile(`@@[a-z]+`)
	linux := sections.FindAllString(gpuProcsScript, -1)
	windows := sections.FindAllString(gpuProcsWinScript, -1)
	if !slices.Equal(linux, windows) {
		t.Errorf("gpuprocs.sh prints %v; gpuprocs.ps1 prints %v", linux, windows)
	}
}

// deadlineClients notes how long a call has left and stops it before any
// SSH session is opened.
type deadlineClients struct{ left *time.Duration }

func (d deadlineClients) Client(string) (*ssh.Client, error) { return nil, errors.New("offline") }

func (d deadlineClients) WaitPlatform(ctx context.Context, _ string) platform.OS {
	if dl, ok := ctx.Deadline(); ok {
		*d.left = time.Until(dl)
	}
	return platform.Linux
}

// A wedged driver can block nvidia-smi for good: the call must give up, so
// the table shows an error and its SSH session is closed, instead of
// piling up sessions on every visit to the GPU tab.
func TestGPUProcessesGiveUpOnAHungNvidiaSMI(t *testing.T) {
	var left time.Duration
	svc := New(deadlineClients{&left}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), Options{})
	t.Cleanup(svc.Close)
	if _, err := svc.GPUProcesses(context.Background(), "s1"); err == nil {
		t.Fatal("offline client accepted")
	}
	if left <= 0 || left > 30*time.Second {
		t.Errorf("time left = %v; want a deadline of at most 30s", left)
	}
}
