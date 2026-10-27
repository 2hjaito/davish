package app

import "davish/internal/sysmon"

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
