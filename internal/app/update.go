package app

import (
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"davish/internal/update"
)

// UpdateCheck asks GitHub for the latest release (docs/adr/0009-self-update.md).
func (a *App) UpdateCheck() (update.Status, error) {
	return a.updates.Check(a.ctx)
}

// UpdateInstall downloads, verifies and swaps in version; progress arrives
// as update:progress. UpdateRestart then switches to it.
func (a *App) UpdateInstall(version string) error {
	return a.updates.Install(a.ctx, version)
}

// UpdateRestart starts the installed version and quits this one.
func (a *App) UpdateRestart() error {
	if err := a.updates.Restart(); err != nil {
		return err
	}
	runtime.Quit(a.ctx)
	return nil
}

// UpdateVersion is the running version.
func (a *App) UpdateVersion() string {
	return a.updates.Current()
}
