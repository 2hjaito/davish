package sysmon

import "time"

// Thresholds decide when a reading is worth a notification.
type Thresholds struct {
	MemPct  float64
	DiskPct float64 // per mount
	// For is how long a reading must stay over its threshold, so a short
	// spike (a build, a log rotation) does not notify.
	For time.Duration
}

// DefaultThresholds are used until Settings can change them (M10).
var DefaultThresholds = Thresholds{MemPct: 90, DiskPct: 85, For: time.Minute}

// Alert kinds; the frontend writes the text for each, as with apperr codes.
const (
	AlertMemHigh  = "mem-high"
	AlertDiskHigh = "disk-high"
)

// Alert is the notify:new payload for a reading that stayed over its
// threshold. ID is stable per server and reading, so a repeat replaces the
// earlier notification instead of piling up.
type Alert struct {
	ID        string  `json:"id"`
	Level     string  `json:"level"`
	ServerID  string  `json:"serverID"`
	Kind      string  `json:"kind"`
	Value     float64 `json:"value"`
	Threshold float64 `json:"threshold"`
	Mount     string  `json:"mount,omitempty"`
}

// rearmMargin is how far under its threshold a reading must fall before it
// can notify again; without it a disk sitting at 85% would notify every
// time it ticks between 84 and 85.
const rearmMargin = 5

// alerter tracks, per reading, since when it has been over its threshold.
// Time is the server's uptime, which keeps it independent of SSH latency
// and lets tests drive it directly.
type alerter struct {
	th   Thresholds
	over map[string]*overState // "mem" or "disk:<path>"
}

type overState struct {
	since float64 // uptime when the reading went over
	fired bool
}

func newAlerter(th Thresholds) *alerter {
	return &alerter{th: th, over: map[string]*overState{}}
}

// resume forgets readings that were over but had not notified yet. It runs
// when a collector starts, because the time it was not watching does not
// count towards For. Fired readings stay fired, so switching servers back
// and forth does not repeat a notification.
func (a *alerter) resume() {
	for k, st := range a.over {
		if !st.fired {
			delete(a.over, k)
		}
	}
}

// check returns the alerts that become due with t.
func (a *alerter) check(t Tick) []Alert {
	var out []Alert
	if t.MemTotal > 0 {
		pct := float64(t.MemUsed) / float64(t.MemTotal) * 100
		if a.due("mem", pct, a.th.MemPct, t.Uptime) {
			out = append(out, Alert{
				ID: t.ServerID + ":mem", Kind: AlertMemHigh,
				Value: pct, Threshold: a.th.MemPct,
			})
		}
	}
	for _, m := range t.Mounts {
		key := "disk:" + m.Path
		if a.due(key, m.Pct, a.th.DiskPct, t.Uptime) {
			out = append(out, Alert{
				ID: t.ServerID + ":" + key, Kind: AlertDiskHigh,
				Value: m.Pct, Threshold: a.th.DiskPct, Mount: m.Path,
			})
		}
	}
	for i := range out {
		out[i].Level = "warn"
		out[i].ServerID = t.ServerID
	}
	return out
}

// due records value for key and reports whether it should notify now.
func (a *alerter) due(key string, value, threshold, now float64) bool {
	if threshold <= 0 {
		return false
	}
	st, ok := a.over[key]
	switch {
	case value >= threshold && !ok:
		fire := a.th.For <= 0
		a.over[key] = &overState{since: now, fired: fire}
		return fire
	case value >= threshold:
		if now < st.since { // rebooted: uptime started over
			st.since = now
		}
		if st.fired || now-st.since < a.th.For.Seconds() {
			return false
		}
		st.fired = true
		return true
	case ok && (!st.fired || value < threshold-rearmMargin):
		delete(a.over, key)
	}
	return false
}
