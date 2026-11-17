package app

import (
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"davish/internal/audit"
)

// AuditEntries returns the commands the app ran on servers, oldest first.
// New ones arrive as audit:entry.
func (a *App) AuditEntries() []audit.Entry {
	return a.audit.Entries()
}

// AuditScript is a bash script that runs the chosen entries again over
// ssh (docs/adr/0022); failed keeps those that exited non-zero.
func (a *App) AuditScript(ids []uint64, failed bool) (string, error) {
	servers, err := a.store.Servers(a.ctx)
	if err != nil {
		return "", err
	}
	opt := audit.ScriptOptions{Failed: failed, Now: time.Now()}
	return audit.Script(a.audit.Pick(ids), audit.Targets(servers), opt), nil
}

// AuditSaveScript asks where to save that script and writes it; "" when
// the user cancels.
func (a *App) AuditSaveScript(ids []uint64, failed bool, title string) (string, error) {
	if err := a.lock.Check(); err != nil {
		return "", err
	}
	script, err := a.AuditScript(ids, failed)
	if err != nil {
		return "", err
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           title,
		DefaultFilename: "davish-commands.sh",
	})
	if err != nil || path == "" {
		return "", err
	}
	return path, audit.SaveScript(path, script)
}
