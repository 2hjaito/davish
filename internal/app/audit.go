package app

import "davish/internal/audit"

// AuditEntries returns the commands the app ran on servers, oldest first.
// New ones arrive as audit:entry.
func (a *App) AuditEntries() []audit.Entry {
	return a.audit.Entries()
}
