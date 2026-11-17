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

// LogsSearch returns the latest lines of src's whole history that contain
// q.Query; token names it for LogsCancelSearch.
func (a *App) LogsSearch(serverID string, src logs.Source, q logs.SearchQuery, token string) (logs.SearchResult, error) {
	if err := a.lock.Check(); err != nil {
		return logs.SearchResult{}, err
	}
	return a.logSearch.Search(a.ctx, serverID, src, q, token)
}

// LogsCancelSearch stops a running LogsSearch.
func (a *App) LogsCancelSearch(token string) {
	a.logSearch.Cancel(token)
}

// LogsAround returns count lines of a log file from line from, to read
// around a search result without loading the file.
func (a *App) LogsAround(serverID, file string, from, count int) (logs.Around, error) {
	if err := a.lock.Check(); err != nil {
		return logs.Around{}, err
	}
	return a.logSearch.Around(a.ctx, serverID, file, from, count)
}
