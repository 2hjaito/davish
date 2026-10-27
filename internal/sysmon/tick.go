package sysmon

import "math"

// Tick is the sysmon:tick payload. Percentages run from 0 to 100, memory and
// totals are bytes, rates are bytes per second.
type Tick struct {
	ServerID string  `json:"serverID"`
	CPU      float64 `json:"cpu"` // all cores together
	MemUsed  uint64  `json:"memUsed"`
	MemTotal uint64  `json:"memTotal"`
	DiskPct  float64 `json:"diskPct"` // filesystem of /
	RxBps    float64 `json:"rxBps"`
	TxBps    float64 `json:"txBps"`
	Load1    float64 `json:"load1"`
	Uptime   float64 `json:"uptime"` // seconds

	Load5    float64   `json:"load5"`
	Load15   float64   `json:"load15"`
	Procs    int       `json:"procs"`
	CPUSplit CPUSplit  `json:"cpuSplit"`
	Cores    []float64 `json:"cores"`

	MemCache  uint64 `json:"memCache"`
	MemFree   uint64 `json:"memFree"`
	SwapTotal uint64 `json:"swapTotal"`
	SwapUsed  uint64 `json:"swapUsed"`

	ReadBps  float64    `json:"readBps"`
	WriteBps float64    `json:"writeBps"`
	Disks    []DiskRate `json:"disks"`
	Mounts   []Mount    `json:"mounts"`
	Ifaces   []NetRate  `json:"ifaces"`
	GPUs     []GPU      `json:"gpus"`
}

// CPUSplit is where the busy time went, as percent of all cores.
type CPUSplit struct {
	User   float64 `json:"user"`   // user and nice
	System float64 `json:"system"` // kernel and interrupts
	IOWait float64 `json:"iowait"`
	Steal  float64 `json:"steal"` // taken by the hypervisor
}

// DiskRate is one disk's throughput.
type DiskRate struct {
	Name     string  `json:"name"`
	ReadBps  float64 `json:"readBps"`
	WriteBps float64 `json:"writeBps"`
}

// NetRate is one interface's throughput and totals since boot.
type NetRate struct {
	Name    string  `json:"name"`
	RxBps   float64 `json:"rxBps"`
	TxBps   float64 `json:"txBps"`
	RxTotal uint64  `json:"rxTotal"`
	TxTotal uint64  `json:"txTotal"`
}

// Rate turns two consecutive snapshots into a Tick. The server's own uptime
// is the clock, so SSH latency and batching do not skew the rates.
func Rate(serverID string, prev, cur Snapshot) Tick {
	m := cur.Mem
	t := Tick{
		ServerID: serverID,
		// Same as `free`: used is what is not available to new programs.
		MemUsed:   m.Total - m.Available,
		MemTotal:  m.Total,
		MemCache:  m.Cache,
		MemFree:   m.Free,
		SwapTotal: m.SwapTotal,
		SwapUsed:  m.SwapTotal - m.SwapFree,
		DiskPct:   cur.Mounts[0].Pct,
		Mounts:    cur.Mounts,
		Load1:     cur.Load1,
		Load5:     cur.Load5,
		Load15:    cur.Load15,
		Procs:     cur.Procs,
		Uptime:    cur.Uptime,
		GPUs:      cur.GPUs,
		Cores:     make([]float64, len(cur.Cores)),
		Ifaces:    make([]NetRate, 0, len(cur.Ifaces)),
		Disks:     make([]DiskRate, 0, len(cur.Disks)),
	}
	t.CPU, t.CPUSplit = cpuBusy(prev.CPU, cur.CPU)
	for i, c := range cur.Cores {
		if i < len(prev.Cores) {
			t.Cores[i], _ = cpuBusy(prev.Cores[i], c)
		}
	}

	dt := cur.Uptime - prev.Uptime
	perSec := func(prev, cur uint64) float64 {
		if dt <= 0 {
			return 0
		}
		return float64(delta(prev, cur)) / dt
	}
	prevNet := map[string]NetCounters{}
	for _, n := range prev.Ifaces {
		prevNet[n.Name] = n
	}
	for _, n := range cur.Ifaces {
		r := NetRate{Name: n.Name, RxTotal: n.Rx, TxTotal: n.Tx}
		if p, ok := prevNet[n.Name]; ok {
			r.RxBps, r.TxBps = perSec(p.Rx, n.Rx), perSec(p.Tx, n.Tx)
		}
		t.RxBps += r.RxBps
		t.TxBps += r.TxBps
		t.Ifaces = append(t.Ifaces, r)
	}
	prevDisk := map[string]DiskCounters{}
	for _, d := range prev.Disks {
		prevDisk[d.Name] = d
	}
	for _, d := range cur.Disks {
		r := DiskRate{Name: d.Name}
		if p, ok := prevDisk[d.Name]; ok {
			r.ReadBps, r.WriteBps = perSec(p.Read, d.Read), perSec(p.Written, d.Written)
		}
		t.ReadBps += r.ReadBps
		t.WriteBps += r.WriteBps
		t.Disks = append(t.Disks, r)
	}
	return t
}

// cpuBusy is the busy percent between two readings, and how it splits.
func cpuBusy(prev, cur CPUTimes) (float64, CPUSplit) {
	total := delta(prev.Total(), cur.Total())
	if total == 0 {
		return 0, CPUSplit{}
	}
	pct := func(d uint64) float64 { return round1(100 * float64(min(d, total)) / float64(total)) }
	busy := total - min(delta(prev.idle(), cur.idle()), total)
	return pct(busy), CPUSplit{
		User:   pct(delta(prev.User+prev.Nice, cur.User+cur.Nice)),
		System: pct(delta(prev.System+prev.IRQ+prev.SoftIRQ, cur.System+cur.IRQ+cur.SoftIRQ)),
		IOWait: pct(delta(prev.IOWait, cur.IOWait)),
		Steal:  pct(delta(prev.Steal, cur.Steal)),
	}
}

// delta is zero when a counter went backwards: a reboot or an interface
// that disappeared, not negative traffic.
func delta(prev, cur uint64) uint64 {
	if cur < prev {
		return 0
	}
	return cur - prev
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}
