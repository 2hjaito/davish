package sysmon

import (
	"regexp"
	"strconv"
	"strings"
)

// parseDRMGPUs reads the "@@gpudrm" lines of collect.sh and collect.ps1, one
// per AMD or Intel card (or any card nvidia-smi does not cover):
//
//	slot|vendor|driver|busy %|vram used B|vram total B|temp m°C|power µW|product name
//
// A field the driver does not give is empty and becomes -1. names holds
// lspci -mm lines, which give a card without a product name a model. The
// GPUs are numbered from first, after those nvidia-smi numbered, so the
// index stays the same between snapshots and matches gpuprocs.
func parseDRMGPUs(lines, names []string, first int) []GPU {
	byslot := pciNames(names)
	var out []GPU
	for _, line := range lines {
		f := strings.SplitN(line, "|", 9)
		if len(f) < 9 {
			continue
		}
		slot, vendor, driver := f[0], f[1], f[2]
		name := strings.TrimSpace(f[8])
		if name == "" {
			name = byslot[slot]
		}
		if name == "" {
			name = vendorName(vendor) + " GPU"
			if driver != "" {
				name += " (" + driver + ")"
			}
		}
		g := GPU{
			Index:    first + len(out),
			Name:     name,
			Util:     gpuValue(f[3], 1),
			MemUsed:  gpuValue(f[4], 1),
			MemTotal: gpuValue(f[5], 1),
			Temp:     gpuValue(f[6], 0.001),
			Power:    gpuValue(f[7], 1e-6),
		}
		out = append(out, g)
	}
	return out
}

// quoted picks the fields of a lspci -mm line.
var quoted = regexp.MustCompile(`"([^"]*)"`)

// brackets is the part of a PCI name in square brackets, the marketing name.
var brackets = regexp.MustCompile(`\[([^\]]+)\]`)

// pciNames maps a PCI slot to "AMD Radeon RX 470/480/570/580" from lspci -mm
// lines: slot, class, vendor, device, ...
func pciNames(lines []string) map[string]string {
	out := map[string]string{}
	for _, line := range lines {
		slot, rest, ok := strings.Cut(strings.TrimSpace(line), " ")
		if !ok {
			continue
		}
		q := quoted.FindAllStringSubmatch(rest, -1)
		if len(q) < 3 {
			continue
		}
		vendor, device := q[1][1], q[2][1]
		if m := brackets.FindStringSubmatch(device); m != nil {
			device = m[1]
		}
		short := vendor
		switch {
		case strings.Contains(vendor, "Advanced Micro Devices"), strings.Contains(vendor, "AMD"):
			short = "AMD"
		case strings.Contains(vendor, "Intel"):
			short = "Intel"
		}
		if !strings.HasPrefix(device, short) {
			device = short + " " + device
		}
		out[slot] = device
	}
	return out
}

// vendorName is a PCI vendor ID as a word.
func vendorName(id string) string {
	switch strings.ToLower(id) {
	case "0x1002", "1002":
		return "AMD"
	case "0x8086", "8086":
		return "Intel"
	case "0x10de", "10de":
		return "NVIDIA"
	}
	if _, err := strconv.ParseUint(strings.TrimPrefix(id, "0x"), 16, 32); err == nil {
		return "GPU " + id
	}
	return "GPU"
}
