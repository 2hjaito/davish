package app

import "davish/internal/logs"

// LogsStart follows a journal unit, a file or a container and returns the
// stream ID with its sudo form; lines arrive as logs:line, the end as logs:end.
func (a *App) LogsStart(serverID string, src logs.Source) (logs.Started, error) {
	return a.logs.Start(a.ctx, serverID, src)
}

// LogsStop ends a stream.
func (a *App) LogsStop(streamID string) {
	a.logs.Stop(streamID)
}
