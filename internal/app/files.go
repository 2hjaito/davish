package app

import "davish/internal/fs"

// FilesHome returns the directory the Files app opens first.
func (a *App) FilesHome(serverID string) (string, error) {
	return a.files.Home(a.ctx, serverID)
}

// FilesList returns the entries of dir sorted by name.
func (a *App) FilesList(serverID, dir string) ([]fs.Entry, error) {
	return a.files.List(a.ctx, serverID, dir)
}

// FilesStat describes path without following a final link.
func (a *App) FilesStat(serverID, path string) (fs.Entry, error) {
	return a.files.Stat(a.ctx, serverID, path)
}

// FilesRead returns a file's content (base64 in JSON), up to fs.MaxRead.
func (a *App) FilesRead(serverID, path string) (fs.File, error) {
	return a.files.Read(a.ctx, serverID, path)
}

// FilesWrite replaces a file atomically, keeping its mode and owner. The
// UI must have confirmed before overwriting an existing file.
func (a *App) FilesWrite(serverID, path string, data []byte) (fs.Entry, error) {
	return a.files.Write(a.ctx, serverID, path, data)
}

// FilesMkdir creates a directory.
func (a *App) FilesMkdir(serverID, path string) error {
	return a.files.Mkdir(a.ctx, serverID, path)
}

// FilesRename moves from to to without replacing anything.
func (a *App) FilesRename(serverID, from, to string) error {
	return a.files.Rename(a.ctx, serverID, from, to)
}

// FilesPlanRemove returns the exact command a delete would run, for the
// confirmation dialog.
func (a *App) FilesPlanRemove(serverID, path string) (fs.RemovePlan, error) {
	return a.files.PlanRemove(a.ctx, serverID, path)
}

// FilesRemove runs the confirmed command from FilesPlanRemove.
func (a *App) FilesRemove(serverID, path, command string) error {
	return a.files.Remove(a.ctx, serverID, path, command)
}

// FilesPlanCompress returns the archive name and the exact command that
// compressing path as format would run.
func (a *App) FilesPlanCompress(serverID, path, format string) (fs.CompressPlan, error) {
	return a.files.PlanCompress(a.ctx, serverID, path, format)
}

// FilesCompress runs the command from FilesPlanCompress.
func (a *App) FilesCompress(serverID, path, format, command string) (fs.CompressPlan, error) {
	return a.files.Compress(a.ctx, serverID, path, format, command)
}

// FilesSave writes a file from the Editor atomically, keeping a backup of
// the previous version when opts.Backup is set.
func (a *App) FilesSave(serverID, path string, data []byte, opts fs.SaveOptions) (fs.SaveResult, error) {
	return a.files.Save(a.ctx, serverID, path, data, opts)
}
