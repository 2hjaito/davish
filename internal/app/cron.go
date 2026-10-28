package app

import (
	"time"

	"davish/internal/apperr"
	"davish/internal/cron"
)

// CronList returns the user's cron jobs and, read-only, the system ones,
// with their next runs.
func (a *App) CronList(serverID string) (cron.Table, error) {
	return a.cron.List(a.ctx, serverID)
}

// CronPlan works out a change to the user's crontab for the confirmation,
// without making it.
func (a *App) CronPlan(serverID string, change cron.Change) (cron.Plan, error) {
	return a.cron.Plan(a.ctx, serverID, change)
}

// CronApply makes a confirmed change, if the crontab is still at version.
func (a *App) CronApply(serverID string, change cron.Change, version string) error {
	return a.cron.Apply(a.ctx, serverID, change, version)
}

// CronRuns returns the system log lines of a job's latest runs.
func (a *App) CronRuns(serverID, command string) (cron.Runs, error) {
	return a.cron.Runs(a.ctx, serverID, command)
}

// CronPreview parses a schedule as the user types it and lists its next
// runs in zone; it does not touch the server.
func (a *App) CronPreview(schedule, zone string) (cron.Preview, error) {
	p, err := cron.PreviewSchedule(schedule, zone, time.Now())
	if err != nil {
		return cron.Preview{}, apperr.New(apperr.InvalidInput, "invalid schedule")
	}
	return p, nil
}
