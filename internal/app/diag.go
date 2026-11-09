package app

import (
	goruntime "runtime"

	"davish/internal/apperr"
	"davish/internal/desktop"
	"davish/internal/diag"
)

// DiagRun finds out why a server does not answer commands.
func (a *App) DiagRun(serverID string) diag.Report {
	return a.diag.Diagnose(a.ctx, serverID)
}

// SupportInfo is what a user sends when asking for help.
type SupportInfo struct {
	Version string `json:"version"`
	OS      string `json:"os"`
	LogPath string `json:"logPath"`
	// LogTail is the end of the log file: commands, errors, connection
	// states. It never holds secrets (docs/SECURITY.md).
	LogTail string `json:"logTail"`
}

// DiagSupportInfo gathers the version, platform and recent log.
func (a *App) DiagSupportInfo() SupportInfo {
	info := SupportInfo{Version: a.version, OS: goruntime.GOOS + "/" + goruntime.GOARCH}
	if a.logFile != nil {
		info.LogPath = a.logFile.Path()
		info.LogTail = a.logFile.Tail(200)
	}
	return info
}

// DiagRevealLog shows the log file in Finder, Explorer or the Linux file
// manager, for attaching it to a bug report.
func (a *App) DiagRevealLog() error {
	if err := a.lock.Check(); err != nil {
		return err
	}
	if a.logFile == nil {
		return apperr.New(apperr.NotFound, "the log file could not be opened")
	}
	return desktop.Reveal(a.ctx, a.logFile.Path())
}

// DiagFrontendError writes an error the interface hit to the log file, so
// "Sao chép chi tiết" carries it to whoever helps. msg is the error's text
// and stack, never user data.
func (a *App) DiagFrontendError(msg string) {
	if len(msg) > 2000 {
		msg = msg[:2000]
	}
	a.log.Warn("frontend error", "err", msg)
}
