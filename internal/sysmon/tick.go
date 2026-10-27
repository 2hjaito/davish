package sysmon

import "math"

// Tick is the sysmon:tick payload.
type Tick struct {
	ServerID string  `json:"serverID"`
	CPU      float64 `json:"cpu"` // percent of all cores, 0 to 100
	MemUsed  uint64  `json:"memUsed"`
	MemTotal uint64  `json:"memTotal"`
	DiskPct  float64 `json:"diskPct"`
	RxBps    float64 `json:"rxBps"` // bytes per second
	TxBps    float64 `json:"txBps"`
	Load1    float64 `json:"load1"`
	Uptime   float64 `json:"uptime"` // seconds
}

// Rate turns two consecutive snapshots into a Tick. The server's own uptime
// is the clock, so SSH latency and batching do not skew the rates.
func Rate(serverID string, prev, cur Snapshot) Tick {
	t := Tick{
		ServerID: serverID,
		// Same as `free`: used is what is not available to new programs.
		MemUsed:  cur.MemTotal - cur.MemAvailable,
		MemTotal: cur.MemTotal,
		DiskPct:  diskPct(cur),
		Load1:    cur.Load1,
		Uptime:   cur.Uptime,
	}
	if total := delta(prev.CPUTotal, cur.CPUTotal); total > 0 {
		busy := total - min(delta(prev.CPUIdle, cur.CPUIdle), total)
		t.CPU = round1(100 * float64(busy) / float64(total))
	}
	if dt := cur.Uptime - prev.Uptime; dt > 0 {
		t.RxBps = float64(delta(prev.RxBytes, cur.RxBytes)) / dt
		t.TxBps = float64(delta(prev.TxBytes, cur.TxBytes)) / dt
	}
	return t
}

// delta is zero when a counter went backwards: a reboot or an interface
// that disappeared, not negative traffic.
func delta(prev, cur uint64) uint64 {
	if cur < prev {
		return 0
	}
	return cur - prev
}

// diskPct rounds up like df's Capacity column, which excludes the blocks
// reserved for root from the total.
func diskPct(s Snapshot) float64 {
	usable := s.DiskUsed + s.DiskAvail
	if usable == 0 {
		return 0
	}
	return math.Ceil(100 * float64(s.DiskUsed) / float64(usable))
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}
