package sysmon

// Health alert kinds, next to AlertMemHigh and AlertDiskHigh.
const (
	AlertServiceFailed       = "service-failed"
	AlertContainerRestarting = "container-restarting"
)

// A container that restarted restartBurst times within restartWindow
// seconds is in a restart loop, even if docker shows it running at the
// moment of the check.
const (
	restartBurst  = 3
	restartWindow = 5 * 60
)

// healthWatch remembers what it already notified, per server, so a unit
// that stays failed notifies once; it notifies again only after recovering.
type healthWatch struct {
	failed     map[string]bool // units notified and still failed
	containers map[string]*containerWatch
}

type containerWatch struct {
	samples []restartSample // within restartWindow, oldest first
	fired   bool
	// troubled is the last uptime it restarted or was not running.
	troubled float64
}

type restartSample struct {
	at       float64 // server uptime
	restarts int
}

func newHealthWatch() *healthWatch {
	return &healthWatch{failed: map[string]bool{}, containers: map[string]*containerWatch{}}
}

// resume drops restart history from before a pause in watching, which
// would make an old burst look recent. What was notified stays notified.
func (w *healthWatch) resume() {
	for _, c := range w.containers {
		c.samples = nil
	}
}

// check returns the alerts h brings, at server uptime now.
func (w *healthWatch) check(serverID string, now float64, h *Health) []Alert {
	var out []Alert

	failed := map[string]bool{}
	for _, u := range h.Failed {
		failed[u] = true
		if !w.failed[u] {
			w.failed[u] = true
			out = append(out, Alert{
				ID: serverID + ":svc:" + u, Level: "error", ServerID: serverID,
				Kind: AlertServiceFailed, Target: u,
			})
		}
	}
	for u := range w.failed {
		if !failed[u] {
			delete(w.failed, u) // recovered: may notify again
		}
	}

	present := map[string]bool{}
	for _, c := range h.Containers {
		present[c.Name] = true
		cw := w.containers[c.Name]
		if cw == nil {
			cw = &containerWatch{}
			w.containers[c.Name] = cw
		}
		if n := len(cw.samples); c.Status != "running" || (n > 0 && c.Restarts > cw.samples[n-1].restarts) {
			cw.troubled = now
		}
		cw.samples = append(cw.samples, restartSample{at: now, restarts: c.Restarts})
		for len(cw.samples) > 1 && now-cw.samples[0].at > restartWindow {
			cw.samples = cw.samples[1:]
		}
		burst := c.Restarts - cw.samples[0].restarts
		looping := c.Status == "restarting" || burst >= restartBurst
		switch {
		case looping && !cw.fired:
			cw.fired = true
			out = append(out, Alert{
				ID: serverID + ":ctr:" + c.Name, Level: "warn", ServerID: serverID,
				Kind: AlertContainerRestarting, Target: c.Name, Value: float64(c.Restarts),
			})
		case cw.fired && !looping && now-cw.troubled >= restartWindow:
			cw.fired = false // ran a whole window without trouble
		}
	}
	for name := range w.containers {
		if !present[name] {
			delete(w.containers, name)
		}
	}
	return out
}
