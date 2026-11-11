package sysmon

import (
	"davish/internal/platform"
)

// States of Status.
const (
	// StateUnsupported: Monitor has no collector for the server's OS.
	StateUnsupported = "unsupported"
	// StateError: the collector runs but no reading ever arrives, such as
	// PowerShell blocked by policy or /proc missing.
	StateError = "error"
)

// Status is the sysmon:status payload: why a server has no readings. It is
// cleared by the first sysmon:tick that follows.
type Status struct {
	ServerID string      `json:"serverID"`
	State    string      `json:"state"`
	OS       platform.OS `json:"os"`
	// Message is the technical reason, for a bug report.
	Message string `json:"message,omitempty"`
}
