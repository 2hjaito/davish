package app

import "davish/internal/fleet"

// FleetCheck tells how a run of command on n servers must be confirmed.
func (a *App) FleetCheck(command string, n int) fleet.Check {
	return fleet.CheckCommand(command, n)
}

// FleetPlanUpdate builds each server's package update command (sudo -n).
func (a *App) FleetPlanUpdate(serverIDs []string) (fleet.Update, error) {
	if err := a.lock.Check(); err != nil {
		return fleet.Update{}, err
	}
	return fleet.PlanUpdate(a.ctx, a.packages, serverIDs), nil
}

// FleetStart runs the items, each in a terminal session of cols×rows;
// progress arrives as fleet:job events.
func (a *App) FleetStart(items []fleet.Item, cols, rows int) (FleetRun, error) {
	if err := a.lock.Check(); err != nil {
		return FleetRun{}, err
	}
	id, jobs, err := a.fleet.Start(a.ctx, items, cols, rows)
	return FleetRun{ID: id, Jobs: jobs}, err
}

// FleetRun is a started run.
type FleetRun struct {
	ID   string      `json:"id"`
	Jobs []fleet.Job `json:"jobs"`
}

// FleetStop stops a run: queued servers do not start, running ones close.
func (a *App) FleetStop(runID string) {
	a.fleet.Stop(runID)
}
