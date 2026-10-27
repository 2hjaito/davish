package app

// MonitorWatch starts sysmon:tick events for the server. The menu bar and
// each Monitor window call it once and pair it with MonitorUnwatch.
func (a *App) MonitorWatch(serverID string) {
	a.sysmon.Watch(serverID)
}

// MonitorUnwatch drops one MonitorWatch; collection stops with the last one.
func (a *App) MonitorUnwatch(serverID string) {
	a.sysmon.Unwatch(serverID)
}
