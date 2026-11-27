package sysmon

import (
	"cmp"
	"context"
	_ "embed"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"davish/internal/apperr"
)

//go:embed gpuprocs.sh
var gpuProcsScript string

// gpuProcsTimeout bounds one reading. A wedged driver (an Xid error) can
// block nvidia-smi for good; without a limit each visit to the GPU tab would
// leave another SSH session hanging. Generous, as nvidia-smi without
// persistence mode can take seconds per call on a many-GPU server.
const gpuProcsTimeout = 30 * time.Second

// GPUProcess is one row of the Monitor's GPU process table, like a row of
// nvtop: a process on one GPU, as nvidia-smi sees it, with what /proc says
// about it. A reading nobody reports is -1.
type GPUProcess struct {
	GPU       int     `json:"gpu"` // nvidia-smi's index
	PID       int     `json:"pid"`
	Type      string  `json:"type"`      // "C" compute, "G" graphics, "C+G", as pmon prints it
	GPUUtil   float64 `json:"gpuUtil"`   // percent of the GPU's SMs
	GPUMem    float64 `json:"gpuMem"`    // bytes
	GPUMemPct float64 `json:"gpuMemPct"` // of that GPU's memory
	User      string  `json:"user"`      // empty when /proc does not show the PID
	Name      string  `json:"name"`
	Command   string  `json:"command"`
	CPU       float64 `json:"cpu"` // percent of one core, as in top
	Mem       float64 `json:"mem"` // resident bytes
}

// GPUProcessList is every process on the server's NVIDIA GPUs, the most GPU
// memory first. A process on two GPUs has a row on each.
type GPUProcessList struct {
	Procs []GPUProcess `json:"procs"`
}

// GPUProcesses lists the processes on the server's NVIDIA GPUs, as nvtop
// does. CPU load is measured since the previous call for the server, apart
// from the Processes tab's; the first call falls back to each process's
// average over its life. A server without nvidia-smi has an empty list.
func (s *Service) GPUProcesses(ctx context.Context, serverID string) (GPUProcessList, error) {
	ctx, cancel := context.WithTimeout(ctx, gpuProcsTimeout)
	defer cancel()
	os := s.clients.WaitPlatform(ctx, serverID)
	client, err := s.clients.Client(serverID)
	if err != nil {
		return GPUProcessList{}, err
	}
	tg, ok := targetFor(os)
	if !ok {
		return GPUProcessList{}, apperr.New(apperr.PlatformUnsupported, "GPU processes are not available for this OS")
	}
	sections, err := runScript(ctx, client, tg, tg.gpuProcs)
	if err != nil {
		return GPUProcessList{}, apperr.Wrap(apperr.Internal, "could not read GPU processes", err)
	}
	snap, err := parseGPUProcs(sections)
	if err != nil {
		return GPUProcessList{}, apperr.Wrap(apperr.Internal, "could not read GPU processes", err)
	}

	s.procsMu.Lock()
	prev := s.gpuProcs[serverID]
	s.gpuProcs[serverID] = snap.sample()
	s.procsMu.Unlock()

	return snap.list(prev), nil
}

// gpuProcSnapshot is one parsed run of gpuprocs.sh: the rows nvidia-smi
// gives and, when /proc showed any of their PIDs, the host's side of them.
type gpuProcSnapshot struct {
	host    procSnapshot
	hasHost bool
	rows    []GPUProcess
}

const bytesPerMiB = 1024 * 1024

// parseGPUProcs merges nvidia-smi's two views of GPU processes:
// --query-compute-apps (memory per compute process) and pmon (load, type and
// graphics processes, when the driver supports it). A GPU without
// processes, or pmon failing, is not an error.
func parseGPUProcs(s map[string][]string) (gpuProcSnapshot, error) {
	var g gpuProcSnapshot
	index, total := parseGPUList(s["gpus"])
	apps := parseComputeApps(s["apps"])
	if len(apps) > 0 && len(index) == 0 {
		return g, errors.New("gpuprocs: compute processes without a GPU list")
	}

	type key struct{ gpu, pid int }
	rows := map[key]*GPUProcess{}
	row := func(gpu, pid int) *GPUProcess {
		k := key{gpu, pid}
		if r, ok := rows[k]; ok {
			return r
		}
		r := &GPUProcess{GPU: gpu, PID: pid, Type: "C", GPUUtil: -1, GPUMem: -1, GPUMemPct: -1, CPU: -1, Mem: -1}
		rows[k] = r
		return r
	}
	for _, a := range apps {
		gpu, ok := index[a.uuid]
		if !ok {
			continue
		}
		r := row(gpu, a.pid)
		r.GPUMem = a.mem
		r.Name, r.Command = baseName(a.name), a.name
	}
	for _, p := range parsePmon(s["pmon"]) {
		r := row(p.gpu, p.pid)
		r.Type, r.GPUUtil = p.typ, p.sm
		if r.GPUMem < 0 {
			r.GPUMem = p.fb
		}
		if p.name != "" {
			r.Name = p.name
		}
		if r.Command == "" {
			r.Command = r.Name
		}
	}

	for _, r := range rows {
		if t := total[r.GPU]; r.GPUMem >= 0 && t > 0 {
			r.GPUMemPct = round1(100 * r.GPUMem / t)
		}
		g.rows = append(g.rows, *r)
	}
	slices.SortFunc(g.rows, func(a, b GPUProcess) int {
		return cmp.Or(cmp.Compare(b.GPUMem, a.GPUMem), cmp.Compare(a.GPU, b.GPU), cmp.Compare(a.PID, b.PID))
	})

	// The script only reads /proc for GPU PIDs, so an idle GPU leaves it
	// empty, and parseProcs would call that an error.
	if len(s["stat"]) > 0 {
		if host, err := parseProcs(s); err == nil {
			g.host, g.hasHost = host, true
		}
	}
	return g, nil
}

// list adds the host's side to each row: user, command, CPU load since prev
// and resident memory. A PID /proc did not show (nvidia-smi running in
// another PID namespace, or a process that just ended) keeps nvidia-smi's name.
func (g gpuProcSnapshot) list(prev procSample) GPUProcessList {
	host := map[int]Process{}
	if g.hasHost {
		for _, p := range g.host.rows(prev) {
			host[p.PID] = p
		}
	}
	out := make([]GPUProcess, 0, len(g.rows))
	for _, r := range g.rows {
		if p, ok := host[r.PID]; ok {
			r.User, r.Name, r.Command = p.User, p.Name, p.Command
			r.CPU, r.Mem = p.CPU, float64(p.Mem)
		}
		out = append(out, r)
	}
	return GPUProcessList{Procs: out}
}

// sample is what the next call needs to measure CPU load.
func (g gpuProcSnapshot) sample() procSample {
	if !g.hasHost {
		return procSample{}
	}
	return g.host.sample()
}

// parseGPUList reads "index, uuid, memory.total" lines into UUID → index
// and index → total bytes.
func parseGPUList(lines []string) (map[string]int, map[int]float64) {
	index, total := map[string]int{}, map[int]float64{}
	for _, line := range lines {
		f := splitCSV(line)
		if len(f) < 3 {
			continue
		}
		i, err := strconv.Atoi(f[0])
		if err != nil {
			continue
		}
		index[f[1]] = i
		total[i] = gpuValue(f[2], bytesPerMiB)
	}
	return index, total
}

type computeApp struct {
	uuid string
	pid  int
	mem  float64 // bytes, -1 when the driver does not say (WDDM)
	name string
}

// parseComputeApps reads "gpu_uuid, pid, used_gpu_memory, process_name"
// lines. The name comes last because a path may hold a comma.
func parseComputeApps(lines []string) []computeApp {
	var out []computeApp
	for _, line := range lines {
		f := splitCSV(line)
		if len(f) < 3 {
			continue
		}
		pid, err := strconv.Atoi(f[1])
		if err != nil {
			continue
		}
		a := computeApp{uuid: f[0], pid: pid, mem: gpuValue(f[2], bytesPerMiB)}
		if len(f) > 3 {
			a.name = strings.Join(f[3:], ", ")
		}
		out = append(out, a)
	}
	return out
}

type pmonRow struct {
	gpu, pid int
	typ      string
	sm, fb   float64 // percent and bytes, -1 when pmon shows "-"
	name     string
}

// parsePmon reads `nvidia-smi pmon -c 1 -s um`. Its columns differ between
// drivers (jpg, ofa and ccpm came later), so they are found by the names on
// the first header line. Without that header (pmon unsupported, an error
// message) there is nothing to read. An idle GPU gets a row of "-", skipped.
func parsePmon(lines []string) []pmonRow {
	col := map[string]int{}
	var out []pmonRow
	for _, line := range lines {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "#"); ok {
			if len(col) == 0 && slices.Contains(strings.Fields(rest), "pid") {
				for i, name := range strings.Fields(rest) {
					col[name] = i
				}
			}
			continue
		}
		gi, ok1 := col["gpu"]
		pi, ok2 := col["pid"]
		if !ok1 || !ok2 {
			continue
		}
		f := strings.Fields(line)
		field := func(name string) string {
			if i, ok := col[name]; ok && i < len(f) {
				return f[i]
			}
			return "-"
		}
		if gi >= len(f) || pi >= len(f) {
			continue
		}
		gpu, err1 := strconv.Atoi(f[gi])
		pid, err2 := strconv.Atoi(f[pi])
		if err1 != nil || err2 != nil {
			continue
		}
		r := pmonRow{gpu: gpu, pid: pid, typ: field("type"), sm: gpuValue(field("sm"), 1), fb: gpuValue(field("fb"), bytesPerMiB)}
		if r.typ == "-" {
			r.typ = "C"
		}
		if i, ok := col["command"]; ok && i < len(f) && f[i] != "-" {
			r.name = strings.Join(f[i:], " ")
		}
		out = append(out, r)
	}
	return out
}

func splitCSV(line string) []string {
	f := strings.Split(line, ",")
	for i := range f {
		f[i] = strings.TrimSpace(f[i])
	}
	return f
}

// baseName is the program's file name from a Linux or Windows path.
func baseName(p string) string {
	return p[strings.LastIndexAny(p, `/\`)+1:]
}
