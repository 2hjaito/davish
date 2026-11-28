package sysmon

import (
	"math"
	"strconv"
	"strings"
)

// drmCard is one AMD or Intel card from the "@@drmcards" lines of
// gpuprocs.sh: slot|vendor|vram total. Cards are numbered after the NVIDIA
// ones, in the order collect.sh numbers them.
type drmCard struct {
	slot  string
	index int
	total float64 // VRAM bytes, -1 when the card does not say
}

func parseDRMCards(lines []string, first int) []drmCard {
	var out []drmCard
	for _, line := range lines {
		f := strings.Split(line, "|")
		if len(f) < 3 {
			continue
		}
		out = append(out, drmCard{slot: f[0], index: first + len(out), total: gpuValue(f[2], 1)})
	}
	return out
}

// engineKey is one process on one card.
type engineKey struct{ gpu, pid int }

// drmUse is what one process holds on one card, summed over its clients.
type drmUse struct {
	mem     float64            // VRAM bytes, -1 when the driver does not count it
	engines map[string]float64 // busy time in ns per engine, since the client began
	render  bool               // draws (gfx, render), not only computes
}

// engineSample is the busy time of the last reading, to measure load
// against.
type engineSample struct {
	uptime float64
	ns     map[engineKey]map[string]float64
}

// parseDRMFD reads `grep -H` lines of /proc/PID/fdinfo/N:
//
//	/proc/1234/fdinfo/7:drm-engine-gfx:	123456 ns
//
// A client (one open of the device) can be shared by several descriptors of
// a process, so each counts once, by its drm-client-id. Descriptors with no
// drm-driver are not GPU descriptors.
func parseDRMFD(lines []string, cards []drmCard) map[engineKey]*drmUse {
	type fd struct {
		pid     int
		driver  string
		pdev    string
		client  string
		engines map[string]float64
		mem     float64
		hasMem  bool
	}
	fds := map[string]*fd{}
	var order []string
	for _, line := range lines {
		path, rest, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key, val, ok := strings.Cut(rest, ":")
		if !ok {
			continue
		}
		val = strings.TrimSpace(val)
		f := fds[path]
		if f == nil {
			pid, ok := pidOfFDPath(path)
			if !ok {
				continue
			}
			f = &fd{pid: pid, engines: map[string]float64{}}
			fds[path] = f
			order = append(order, path)
		}
		switch {
		case key == "drm-driver":
			f.driver = val
		case key == "drm-pdev":
			f.pdev = val
		case key == "drm-client-id":
			f.client = val
		case strings.HasPrefix(key, "drm-engine-"):
			if ns, ok := firstNumber(val); ok {
				f.engines[strings.TrimPrefix(key, "drm-engine-")] = ns
			}
		case key == "drm-memory-vram", strings.HasPrefix(key, "drm-total-vram"), strings.HasPrefix(key, "drm-total-local"):
			if n, ok := sizeBytes(val); ok {
				f.mem, f.hasMem = f.mem+n, true
			}
		}
	}

	out := map[engineKey]*drmUse{}
	seen := map[string]bool{}
	for _, path := range order {
		f := fds[path]
		if f.driver == "" {
			continue
		}
		card, ok := cardFor(f.pdev, cards)
		if !ok {
			continue
		}
		if f.client != "" {
			id := strconv.Itoa(f.pid) + "/" + f.pdev + "/" + f.client
			if seen[id] {
				continue
			}
			seen[id] = true
		}
		k := engineKey{gpu: card.index, pid: f.pid}
		u := out[k]
		if u == nil {
			u = &drmUse{mem: -1, engines: map[string]float64{}}
			out[k] = u
		}
		if f.hasMem {
			u.mem = max(u.mem, 0) + f.mem
		}
		for name, ns := range f.engines {
			u.engines[name] += ns
			if name == "gfx" || name == "render" || name == "3d" {
				u.render = true
			}
		}
	}
	return out
}

// cardFor finds a descriptor's card by PCI slot. A kernel that does not print
// drm-pdev gets the card there is, if there is only one.
func cardFor(pdev string, cards []drmCard) (drmCard, bool) {
	for _, c := range cards {
		if c.slot != "" && c.slot == pdev {
			return c, true
		}
	}
	if pdev == "" && len(cards) == 1 {
		return cards[0], true
	}
	return drmCard{}, false
}

// pidOfFDPath is 1234 from ".../1234/fdinfo/7".
func pidOfFDPath(path string) (int, bool) {
	parts := strings.Split(path, "/")
	for i := len(parts) - 2; i > 0; i-- {
		if parts[i] == "fdinfo" {
			pid, err := strconv.Atoi(parts[i-1])
			return pid, err == nil
		}
	}
	return 0, false
}

// firstNumber is the number a "123456 ns" or "37" value starts with.
func firstNumber(s string) (float64, bool) {
	f := strings.Fields(s)
	if len(f) == 0 {
		return 0, false
	}
	n, err := strconv.ParseFloat(f[0], 64)
	return n, err == nil && n >= 0
}

// sizeBytes reads "1024 KiB", the unit the kernel prints for DRM memory.
func sizeBytes(s string) (float64, bool) {
	n, ok := firstNumber(s)
	if !ok {
		return 0, false
	}
	f := strings.Fields(s)
	if len(f) > 1 {
		switch strings.ToLower(f[1]) {
		case "kib":
			n *= 1024
		case "mib":
			n *= 1024 * 1024
		case "gib":
			n *= 1024 * 1024 * 1024
		}
	}
	return n, true
}

// engineLoad is the share of the card's time the process kept its busiest
// engine occupied since prev, in percent; -1 when there is nothing to compare.
func engineLoad(cur, prev map[string]float64, seconds float64) float64 {
	if prev == nil || seconds <= 0 {
		return -1
	}
	var best float64
	for name, ns := range cur {
		if p, ok := prev[name]; ok && ns >= p {
			best = max(best, (ns-p)/(seconds*1e9)*100)
		}
	}
	return math.Round(min(best, 100)*10) / 10
}
