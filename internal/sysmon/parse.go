package sysmon

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

// CPUTimes are the jiffy counters of one /proc/stat cpu line. Guest time is
// already counted in user and nice, so it is left out.
type CPUTimes struct {
	User, Nice, System, Idle, IOWait, IRQ, SoftIRQ, Steal uint64
}

// Total is every jiffy spent in any state.
func (c CPUTimes) Total() uint64 {
	return c.User + c.Nice + c.System + c.Idle + c.IOWait + c.IRQ + c.SoftIRQ + c.Steal
}

func (c CPUTimes) idle() uint64 { return c.Idle + c.IOWait }

// Memory is /proc/meminfo in bytes.
type Memory struct {
	Total, Free, Available uint64
	// Cache is buffers, page cache and reclaimable slab, like free's buff/cache.
	Cache               uint64
	SwapTotal, SwapFree uint64
}

// NetCounters are an interface's cumulative byte counters.
type NetCounters struct {
	Name   string
	Rx, Tx uint64
}

// DiskCounters are a block device's cumulative bytes read and written.
type DiskCounters struct {
	Name          string
	Read, Written uint64
}

// Mount is one filesystem from df, in bytes.
type Mount struct {
	Device string  `json:"device"`
	Path   string  `json:"path"`
	Size   uint64  `json:"size"`
	Used   uint64  `json:"used"`
	Avail  uint64  `json:"avail"`
	Pct    float64 `json:"pct"` // rounded up like df's Capacity
}

// GPU is one NVIDIA GPU from nvidia-smi. Readings the driver does not
// report are -1.
type GPU struct {
	Index    int     `json:"index"`
	Name     string  `json:"name"`
	Util     float64 `json:"util"` // percent
	MemUsed  float64 `json:"memUsed"`
	MemTotal float64 `json:"memTotal"` // bytes
	Temp     float64 `json:"temp"`     // °C
	Power    float64 `json:"power"`    // watts
}

// Snapshot is one reading of collect.sh. Counters are cumulative since boot;
// Rate turns two of them into a Tick.
type Snapshot struct {
	CPU   CPUTimes
	Cores []CPUTimes
	Mem   Memory

	Load1, Load5, Load15 float64
	Procs                int     // processes and threads that exist
	Uptime               float64 // seconds since boot, the clock for every delta

	Ifaces []NetCounters  // interfaces that carry real traffic
	Mounts []Mount        // real filesystems, / first
	Disks  []DiskCounters // whole disks, not partitions
	GPUs   []GPU

	// Health is set on the snapshots that check services and containers,
	// every 30 seconds; nil on the others.
	Health *Health
}

// Health is what can break without a number going up: failed services and
// containers that keep restarting.
type Health struct {
	Failed     []string // failed service units
	Containers []ContainerState
}

// ContainerState is one container as docker inspect saw it.
type ContainerState struct {
	Name     string
	Restarts int
	Status   string // running, restarting, exited…
}

// ErrMalformed marks a snapshot that could not be parsed; the stream after
// it is still usable.
var ErrMalformed = errors.New("malformed snapshot")

// Reader splits the collector's output stream into snapshots.
type Reader struct {
	sc *bufio.Scanner
}

// NewReader reads collect.sh output from r.
func NewReader(r io.Reader) *Reader {
	sc := bufio.NewScanner(r)
	// A /proc/[pid]/stat or nvidia-smi line can outgrow the 64KB default on
	// odd systems; one long line must not end the stream.
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	return &Reader{sc: sc}
}

// Next returns the next snapshot, io.EOF at the end of the stream, or an
// error wrapping ErrMalformed for a snapshot that should be skipped.
func (r *Reader) Next() (Snapshot, error) {
	sections, err := r.sections()
	if err != nil {
		return Snapshot{}, err
	}
	snap, err := parseSections(sections)
	if err != nil {
		return Snapshot{}, fmt.Errorf("%w: %w", ErrMalformed, err)
	}
	return snap, nil
}

// sections reads up to the next @@end and groups the lines by @@section.
func (r *Reader) sections() (map[string][]string, error) {
	sections := map[string][]string{}
	current := ""
	for r.sc.Scan() {
		line := r.sc.Text()
		if name, ok := strings.CutPrefix(line, "@@"); ok {
			if name == "end" {
				return sections, nil
			}
			current = name
			sections[current] = nil
			continue
		}
		if current != "" {
			sections[current] = append(sections[current], line)
		}
	}
	if err := r.sc.Err(); err != nil {
		return nil, fmt.Errorf("read script output: %w", err)
	}
	return nil, io.EOF
}

func parseSections(s map[string][]string) (Snapshot, error) {
	var snap Snapshot
	var err error
	if snap.CPU, snap.Cores, err = parseStat(s["stat"]); err != nil {
		return Snapshot{}, err
	}
	if snap.Mem, err = parseMeminfo(s["meminfo"]); err != nil {
		return Snapshot{}, err
	}
	if snap.Load1, snap.Load5, snap.Load15, snap.Procs, err = parseLoadavg(s["loadavg"]); err != nil {
		return Snapshot{}, err
	}
	if snap.Uptime, err = firstFloat(s["uptime"], "uptime"); err != nil {
		return Snapshot{}, err
	}
	if snap.Ifaces, err = parseNetDev(s["netdev"]); err != nil {
		return Snapshot{}, err
	}
	if snap.Mounts, err = parseDf(s["df"]); err != nil {
		return Snapshot{}, err
	}
	// Disk I/O and GPUs are extras: a container without /sys/block or a
	// driver without a reading must not cost the whole snapshot.
	snap.Disks = parseDiskstats(s["diskstats"], s["block"])
	snap.GPUs = parseGPUs(s["gpu"])
	if failed, ok := s["failed"]; ok {
		snap.Health = parseHealth(failed, s["containers"])
	}
	return snap, nil
}

// parseHealth reads unit names, one per line, and "/name restarts status"
// lines from docker inspect.
func parseHealth(failed, containers []string) *Health {
	h := &Health{}
	for _, l := range failed {
		if u := strings.TrimSpace(l); strings.HasSuffix(u, ".service") {
			h.Failed = append(h.Failed, u)
		}
	}
	for _, l := range containers {
		f := strings.Fields(l)
		if len(f) != 3 {
			continue
		}
		n, err := strconv.Atoi(f[1])
		if err != nil {
			continue
		}
		h.Containers = append(h.Containers, ContainerState{Name: strings.TrimPrefix(f[0], "/"), Restarts: n, Status: f[2]})
	}
	return h
}

// parseStat reads the "cpu" and "cpuN" lines of /proc/stat.
func parseStat(lines []string) (all CPUTimes, cores []CPUTimes, err error) {
	seen := false
	cores = []CPUTimes{}
	for _, line := range lines {
		f := strings.Fields(line)
		if len(f) < 5 || !strings.HasPrefix(f[0], "cpu") {
			return CPUTimes{}, nil, fmt.Errorf("stat: unexpected line %q", line)
		}
		var v [8]uint64
		for i, s := range f[1:min(len(f), 9)] {
			if v[i], err = strconv.ParseUint(s, 10, 64); err != nil {
				return CPUTimes{}, nil, fmt.Errorf("stat: %s field %d: %w", f[0], i+1, err)
			}
		}
		t := CPUTimes{v[0], v[1], v[2], v[3], v[4], v[5], v[6], v[7]}
		if f[0] == "cpu" {
			all, seen = t, true
		} else {
			cores = append(cores, t)
		}
	}
	if !seen {
		return CPUTimes{}, nil, errors.New("stat: no cpu line")
	}
	return all, cores, nil
}

// parseMeminfo reads /proc/meminfo. Kernels before 3.14 lack MemAvailable;
// free + buffers + cache approximates it there.
func parseMeminfo(lines []string) (Memory, error) {
	kb := map[string]uint64{}
	for _, line := range lines {
		key, rest, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		f := strings.Fields(rest)
		if len(f) == 0 {
			continue
		}
		if n, err := strconv.ParseUint(f[0], 10, 64); err == nil {
			kb[key] = n
		}
	}
	total, ok := kb["MemTotal"]
	if !ok || total == 0 {
		return Memory{}, errors.New("meminfo: no MemTotal")
	}
	available, ok := kb["MemAvailable"]
	if !ok {
		available = kb["MemFree"] + kb["Buffers"] + kb["Cached"]
	}
	return Memory{
		Total:     total * 1024,
		Free:      min(kb["MemFree"], total) * 1024,
		Available: min(available, total) * 1024,
		Cache:     min(kb["Buffers"]+kb["Cached"]+kb["SReclaimable"], total) * 1024,
		SwapTotal: kb["SwapTotal"] * 1024,
		SwapFree:  min(kb["SwapFree"], kb["SwapTotal"]) * 1024,
	}, nil
}

// parseLoadavg reads "0.50 0.40 0.30 1/100 42"; the fourth field is
// running/existing scheduling entities.
func parseLoadavg(lines []string) (l1, l5, l15 float64, procs int, err error) {
	if len(lines) == 0 {
		return 0, 0, 0, 0, errors.New("loadavg: missing")
	}
	f := strings.Fields(lines[0])
	if len(f) < 4 {
		return 0, 0, 0, 0, fmt.Errorf("loadavg: unexpected line %q", lines[0])
	}
	var loads [3]float64
	for i := range loads {
		if loads[i], err = strconv.ParseFloat(f[i], 64); err != nil {
			return 0, 0, 0, 0, fmt.Errorf("loadavg: %w", err)
		}
	}
	if _, total, ok := strings.Cut(f[3], "/"); ok {
		procs, _ = strconv.Atoi(total)
	}
	return loads[0], loads[1], loads[2], procs, nil
}

func firstFloat(lines []string, name string) (float64, error) {
	if len(lines) == 0 {
		return 0, fmt.Errorf("%s: missing", name)
	}
	f := strings.Fields(lines[0])
	if len(f) == 0 {
		return 0, fmt.Errorf("%s: empty", name)
	}
	v, err := strconv.ParseFloat(f[0], 64)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", name, err)
	}
	return v, nil
}

// virtualPrefixes are interfaces whose traffic is already counted on a
// physical one (container bridges and veth pairs) or never leaves the host.
var virtualPrefixes = []string{"lo", "veth", "docker", "br-", "virbr", "cni", "flannel", "cali", "kube-"}

// parseNetDev reads /proc/net/dev. The name and the first counter can touch
// ("eth0:123"), so split on the colon.
func parseNetDev(lines []string) ([]NetCounters, error) {
	seen := false
	out := []NetCounters{}
	for _, line := range lines {
		name, rest, ok := strings.Cut(line, ":")
		if !ok {
			continue // the two header lines
		}
		name = strings.TrimSpace(name)
		f := strings.Fields(rest)
		if len(f) < 9 {
			return nil, fmt.Errorf("netdev: short line for %s", name)
		}
		seen = true
		if hasAnyPrefix(name, virtualPrefixes) {
			continue
		}
		rx, err1 := strconv.ParseUint(f[0], 10, 64)
		tx, err2 := strconv.ParseUint(f[8], 10, 64)
		if err := errors.Join(err1, err2); err != nil {
			return nil, fmt.Errorf("netdev: %s: %w", name, err)
		}
		// The kernel creates idle tunnel devices (tunl0, gre0, sit0…) that
		// never carry a byte; listing them only buries the real ones.
		if rx == 0 && tx == 0 {
			continue
		}
		out = append(out, NetCounters{Name: name, Rx: rx, Tx: tx})
	}
	if !seen {
		return nil, errors.New("netdev: no interfaces")
	}
	return out, nil
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// parseDf reads `df -kP`. -P keeps long device names from wrapping, but a
// device name may still hold spaces, so fields are taken from the right.
// Kept are / and the filesystems on real block devices, once each.
func parseDf(lines []string) ([]Mount, error) {
	if len(lines) < 2 {
		return nil, errors.New("df: missing")
	}
	var root *Mount
	var rest []Mount
	seen := map[string]bool{}
	for _, line := range lines[1:] {
		m, err := parseDfLine(line)
		if err != nil {
			return nil, err
		}
		if m.Path == "/" {
			root = &m
			continue
		}
		if !strings.HasPrefix(m.Device, "/dev/") || strings.HasPrefix(m.Device, "/dev/loop") || seen[m.Device] {
			continue
		}
		seen[m.Device] = true
		rest = append(rest, m)
	}
	if root == nil {
		return nil, errors.New("df: no / filesystem")
	}
	out := []Mount{*root}
	for _, m := range rest {
		// Inside a container the host disk shows up again under bind mounts
		// such as /etc/hosts; identical numbers mean the same filesystem.
		if m.Device == root.Device || (m.Size == root.Size && m.Used == root.Used && m.Avail == root.Avail) {
			continue
		}
		out = append(out, m)
	}
	return out, nil
}

func parseDfLine(line string) (Mount, error) {
	f := strings.Fields(line)
	if len(f) < 6 {
		return Mount{}, fmt.Errorf("df: unexpected line %q", line)
	}
	n := len(f)
	var vals [3]uint64
	for i, v := range f[n-5 : n-2] { // size, used, available; capacity and mount follow
		var err error
		if vals[i], err = strconv.ParseUint(v, 10, 64); err != nil {
			return Mount{}, fmt.Errorf("df: %w", err)
		}
	}
	m := Mount{
		Device: strings.Join(f[:n-5], " "),
		Path:   f[n-1],
		Size:   vals[0] * 1024,
		Used:   vals[1] * 1024,
		Avail:  vals[2] * 1024,
	}
	// df's Capacity leaves out the blocks reserved for root.
	if usable := m.Used + m.Avail; usable > 0 {
		m.Pct = math.Ceil(100 * float64(m.Used) / float64(usable))
	}
	return m, nil
}

// pseudoDisks are block devices that are not disks, or whose I/O is already
// counted on the disks underneath (device mapper, software RAID).
var pseudoDisks = []string{"loop", "ram", "zram", "nbd", "dm-", "md", "sr", "fd"}

// sectorSize is the unit of /proc/diskstats, whatever the disk's own.
const sectorSize = 512

// parseDiskstats sums whole disks only: /sys/block lists them, and leaving
// out partitions keeps every byte counted once.
func parseDiskstats(lines, block []string) []DiskCounters {
	whole := map[string]bool{}
	for _, name := range block {
		whole[strings.TrimSpace(name)] = true
	}
	out := []DiskCounters{}
	for _, line := range lines {
		f := strings.Fields(line)
		if len(f) < 10 {
			continue
		}
		name := f[2]
		if !whole[name] || hasAnyPrefix(name, pseudoDisks) {
			continue
		}
		read, err1 := strconv.ParseUint(f[5], 10, 64)
		written, err2 := strconv.ParseUint(f[9], 10, 64)
		if err1 != nil || err2 != nil {
			continue
		}
		out = append(out, DiskCounters{Name: name, Read: read * sectorSize, Written: written * sectorSize})
	}
	return out
}

// parseGPUs reads nvidia-smi's CSV: index, name, util %, memory used and
// total in MiB, temperature, power in W. The name may hold a comma, so the
// numbers are taken from both ends.
func parseGPUs(lines []string) []GPU {
	out := []GPU{}
	for _, line := range lines {
		f := strings.Split(line, ",")
		if len(f) < 7 {
			continue
		}
		for i := range f {
			f[i] = strings.TrimSpace(f[i])
		}
		idx, err := strconv.Atoi(f[0])
		if err != nil {
			continue
		}
		n := len(f)
		const mib = 1024 * 1024
		g := GPU{
			Index:    idx,
			Name:     strings.Join(f[1:n-5], ", "),
			Util:     gpuValue(f[n-5], 1),
			MemUsed:  gpuValue(f[n-4], mib),
			MemTotal: gpuValue(f[n-3], mib),
			Temp:     gpuValue(f[n-2], 1),
			Power:    gpuValue(f[n-1], 1),
		}
		out = append(out, g)
	}
	return out
}

// gpuValue is -1 for "[N/A]", "[Not Supported]" and the like.
func gpuValue(s string, scale float64) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v < 0 {
		return -1
	}
	return v * scale
}
