package app

import (
	"davish/internal/disk"
	"davish/internal/fs"
)

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

// DiskPlanRemove is the delete command for a file or folder found by the
// Disk app; system folders are refused.
func (a *App) DiskPlanRemove(serverID, path string) (fs.RemovePlan, error) {
	return a.disk.PlanRemove(a.ctx, serverID, path)
}

// DiskRemove runs the confirmed command from DiskPlanRemove.
func (a *App) DiskRemove(serverID, path, command string) error {
	if err := a.lock.Check(); err != nil {
		return err
	}
	return a.disk.Remove(a.ctx, serverID, path, command)
}

// DiskPlanTruncate is the command that empties a file, such as a log.
func (a *App) DiskPlanTruncate(path string) (disk.TruncatePlan, error) {
	return disk.PlanTruncate(path)
}

// DiskTruncate runs the confirmed command from DiskPlanTruncate.
func (a *App) DiskTruncate(serverID, path, command string) error {
	if err := a.lock.Check(); err != nil {
		return err
	}
	return a.disk.Truncate(a.ctx, serverID, path, command)
}
