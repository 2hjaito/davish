package app

import "davish/internal/logs"

// LogsStart follows a journal unit, a file or a container and returns the
// stream ID with its sudo form; lines arrive as logs:line, the end as logs:end.
func (a *App) LogsStart(serverID string, src logs.Source) (logs.Started, error) {
	return a.logs.Start(a.ctx, serverID, src)
}

// LogsStartAsRoot follows the same source as root through sudo, never
// asking for the password itself (ADR 0018).
func (a *App) LogsStartAsRoot(serverID string, src logs.Source) (logs.Started, error) {
	if err := a.lock.Check(); err != nil {
		return logs.Started{}, err
	}
	return a.logs.StartAsRoot(a.ctx, serverID, src)
}

// LogsStop ends a stream.
func (a *App) LogsStop(streamID string) {
	a.logs.Stop(streamID)
}
