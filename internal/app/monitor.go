package app

import (
	"time"

	"davish/internal/apperr"
	"davish/internal/sysmon"
)

// MonitorWatch starts sysmon:tick events for the server. The menu bar and
// each Monitor window call it once and pair it with MonitorUnwatch.
func (a *App) MonitorWatch(serverID string) {
	a.sysmon.Watch(serverID)
}

// MonitorUnwatch drops one MonitorWatch; collection stops with the last one.
func (a *App) MonitorUnwatch(serverID string) {
	a.sysmon.Unwatch(serverID)
}

// MonitorProcesses lists the busiest processes by sortBy ("cpu" or "mem").
func (a *App) MonitorProcesses(serverID, sortBy string) (sysmon.ProcessList, error) {
	return a.sysmon.Processes(a.ctx, serverID, sortBy)
}

// MonitorSystemInfo describes the server: OS, kernel, CPU model.
func (a *App) MonitorSystemInfo(serverID string) (sysmon.SystemInfo, error) {
	return a.sysmon.SystemInfo(a.ctx, serverID)
}

// MonitorSetThresholds changes when RAM and disk notify: percents between
// 50 and 99, held for forSec seconds (0 to 3600) before notifying.
func (a *App) MonitorSetThresholds(memPct, diskPct float64, forSec int) error {
	if memPct < 50 || memPct > 99 || diskPct < 50 || diskPct > 99 || forSec < 0 || forSec > 3600 {
		return apperr.New(apperr.InvalidInput, "thresholds out of range")
	}
	a.sysmon.SetThresholds(sysmon.Thresholds{MemPct: memPct, DiskPct: diskPct, For: time.Duration(forSec) * time.Second})
	return nil
}
