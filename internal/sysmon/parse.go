package sysmon

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Snapshot is one reading of collect.sh. Counters are cumulative since boot;
// Rate turns two of them into a Tick.
type Snapshot struct {
	CPUTotal uint64 // jiffies across every state
	CPUIdle  uint64 // idle + iowait jiffies

	MemTotal     uint64 // bytes
	MemAvailable uint64 // bytes

	Load1  float64
	Uptime float64 // seconds since boot, the clock for every delta

	RxBytes uint64 // summed over the interfaces that carry real traffic
	TxBytes uint64

	DiskSize  uint64 // bytes, filesystem of /
	DiskUsed  uint64
	DiskAvail uint64
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
	return &Reader{sc: bufio.NewScanner(r)}
}

// Next returns the next snapshot, io.EOF at the end of the stream, or an
// error wrapping ErrMalformed for a snapshot that should be skipped.
func (r *Reader) Next() (Snapshot, error) {
	sections := map[string][]string{}
	current := ""
	for r.sc.Scan() {
		line := r.sc.Text()
		if name, ok := strings.CutPrefix(line, "@@"); ok {
			if name == "end" {
				snap, err := parseSections(sections)
				if err != nil {
					return Snapshot{}, fmt.Errorf("%w: %w", ErrMalformed, err)
				}
				return snap, nil
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
		return Snapshot{}, fmt.Errorf("read collector output: %w", err)
	}
	return Snapshot{}, io.EOF
}

func parseSections(s map[string][]string) (Snapshot, error) {
	var snap Snapshot
	var err error
	if snap.CPUTotal, snap.CPUIdle, err = parseStat(s["stat"]); err != nil {
		return Snapshot{}, err
	}
	if snap.MemTotal, snap.MemAvailable, err = parseMeminfo(s["meminfo"]); err != nil {
		return Snapshot{}, err
	}
	if snap.Load1, err = firstFloat(s["loadavg"], "loadavg"); err != nil {
		return Snapshot{}, err
	}
	if snap.Uptime, err = firstFloat(s["uptime"], "uptime"); err != nil {
		return Snapshot{}, err
	}
	if snap.RxBytes, snap.TxBytes, err = parseNetDev(s["netdev"]); err != nil {
		return Snapshot{}, err
	}
	if snap.DiskSize, snap.DiskUsed, snap.DiskAvail, err = parseDf(s["df"]); err != nil {
		return Snapshot{}, err
	}
	return snap, nil
}

// parseStat reads the aggregate "cpu" line of /proc/stat. Guest time is
// already counted in user and nice, so only the first eight fields add up.
func parseStat(lines []string) (total, idle uint64, err error) {
	if len(lines) == 0 {
		return 0, 0, errors.New("stat: missing")
	}
	f := strings.Fields(lines[0])
	if len(f) < 5 || f[0] != "cpu" {
		return 0, 0, fmt.Errorf("stat: unexpected line %q", lines[0])
	}
	for i, v := range f[1:min(len(f), 9)] {
		n, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("stat: field %d: %w", i+1, err)
		}
		total += n
		if i == 3 || i == 4 { // idle, iowait
			idle += n
		}
	}
	return total, idle, nil
}

// parseMeminfo returns MemTotal and MemAvailable in bytes. Kernels before
// 3.14 lack MemAvailable; free + buffers + cache approximates it there.
func parseMeminfo(lines []string) (total, available uint64, err error) {
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
		return 0, 0, errors.New("meminfo: no MemTotal")
	}
	available, ok = kb["MemAvailable"]
	if !ok {
		available = kb["MemFree"] + kb["Buffers"] + kb["Cached"]
	}
	return total * 1024, min(available, total) * 1024, nil
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

// parseNetDev sums received and sent bytes over /proc/net/dev. The name and
// the first counter can touch ("eth0:123"), so split on the colon.
func parseNetDev(lines []string) (rx, tx uint64, err error) {
	seen := false
	for _, line := range lines {
		name, rest, ok := strings.Cut(line, ":")
		if !ok {
			continue // the two header lines
		}
		name = strings.TrimSpace(name)
		f := strings.Fields(rest)
		if len(f) < 9 {
			return 0, 0, fmt.Errorf("netdev: short line for %s", name)
		}
		seen = true
		if isVirtual(name) {
			continue
		}
		r, err1 := strconv.ParseUint(f[0], 10, 64)
		t, err2 := strconv.ParseUint(f[8], 10, 64)
		if err := errors.Join(err1, err2); err != nil {
			return 0, 0, fmt.Errorf("netdev: %s: %w", name, err)
		}
		rx += r
		tx += t
	}
	if !seen {
		return 0, 0, errors.New("netdev: no interfaces")
	}
	return rx, tx, nil
}

func isVirtual(name string) bool {
	for _, p := range virtualPrefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

// parseDf reads `df -kP /`: a header, then one line ending in the mount
// point. -P keeps long device names from wrapping onto a second line.
func parseDf(lines []string) (size, used, avail uint64, err error) {
	if len(lines) < 2 {
		return 0, 0, 0, errors.New("df: missing")
	}
	f := strings.Fields(lines[len(lines)-1])
	if len(f) < 6 {
		return 0, 0, 0, fmt.Errorf("df: unexpected line %q", lines[len(lines)-1])
	}
	n := len(f)
	var vals [3]uint64
	for i, v := range f[n-5 : n-2] { // size, used, available; capacity and mount follow
		if vals[i], err = strconv.ParseUint(v, 10, 64); err != nil {
			return 0, 0, 0, fmt.Errorf("df: %w", err)
		}
	}
	return vals[0] * 1024, vals[1] * 1024, vals[2] * 1024, nil
}
