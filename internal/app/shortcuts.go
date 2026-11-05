package app

import "davish/internal/shortcut"

// ShortcutInspect reports what a desktop icon's path is now, and for a
// script the command a run would do.
func (a *App) ShortcutInspect(serverID, path string) (shortcut.Target, error) {
	if err := a.lock.Check(); err != nil {
		return shortcut.Target{}, err
	}
	return a.shortcuts.Inspect(a.ctx, serverID, path)
}

// ShortcutRun returns the terminal line that runs the script at path,
// refused when its content no longer has the given hash.
func (a *App) ShortcutRun(serverID, path, hash string) (shortcut.Run, error) {
	if err := a.lock.Check(); err != nil {
		return shortcut.Run{}, err
	}
	return a.shortcuts.Run(a.ctx, serverID, path, hash)
}
