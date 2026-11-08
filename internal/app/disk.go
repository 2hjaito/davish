package app

import "davish/internal/disk"

// DiskMounts lists the server's filesystems.
func (a *App) DiskMounts(serverID string) ([]disk.Mount, error) {
	return a.disk.Mounts(a.ctx, serverID)
}

// DiskUsage measures the children of one folder; token names it for DiskCancel.
func (a *App) DiskUsage(serverID, dir, token string) (disk.Usage, error) {
	return a.disk.Usage(a.ctx, serverID, dir, token)
}

// DiskOverview measures where a filesystem's space goes; token names it
// for DiskCancel.
func (a *App) DiskOverview(serverID, mount, token string) (disk.Overview, error) {
	return a.disk.Overview(a.ctx, serverID, mount, token)
}

// DiskCancel stops a running DiskUsage or DiskOverview.
func (a *App) DiskCancel(token string) {
	a.disk.Cancel(token)
}

// DiskCleanups lists what could be cleaned and how much it would free.
func (a *App) DiskCleanups(serverID string) ([]disk.Cleanup, error) {
	return a.disk.Cleanups(a.ctx, serverID)
}

// DiskClean runs a cleanup after the user confirmed its exact command.
func (a *App) DiskClean(serverID, id, command string) error {
	if err := a.lock.Check(); err != nil {
		return err
	}
	return a.disk.Clean(a.ctx, serverID, id, command)
}
