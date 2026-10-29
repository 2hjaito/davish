// Package events holds the names of every event emitted from Go to the
// frontend. Keep in sync with frontend/src/lib/api/events.ts.
package events

// Event names, grouped by the service that emits them.
const (
	ConnStatus        = "conn:status"
	ConnHostKeyPrompt = "conn:hostkey-prompt"
	TermData          = "term:data"
	TermExit          = "term:exit"
	SysmonTick        = "sysmon:tick"
	LogsLine          = "logs:line"
	LogsEnd           = "logs:end"
	TransferProgress  = "transfer:progress"
	TransferDone      = "transfer:done"
	NotifyNew         = "notify:new"
	AuditEntry        = "audit:entry"
	LockState         = "lock:state"
	TunnelChanged     = "tunnel:changed"
	UpdateProgress    = "update:progress"
)
