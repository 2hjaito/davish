package app

import "davish/internal/svc"

// ServicesList returns the server's systemd services sorted by name.
func (a *App) ServicesList(serverID string) ([]svc.Unit, error) {
	return a.services.List(a.ctx, serverID)
}

// ServicesStatus returns the detail of one service.
func (a *App) ServicesStatus(serverID, unit string) (svc.Status, error) {
	return a.services.Status(a.ctx, serverID, unit)
}

// ServicesJournal returns a service's latest log lines.
func (a *App) ServicesJournal(serverID, unit string) (svc.Journal, error) {
	return a.services.Journal(a.ctx, serverID, unit)
}

// ServicesPlan returns the exact command an action would run, and whether
// the UI must confirm it first.
func (a *App) ServicesPlan(unit, action string) (svc.Plan, error) {
	return svc.PlanAction(unit, action)
}

// ServicesAct starts, stops or restarts a service. Stop and restart must
// have been confirmed with the command from ServicesPlan.
func (a *App) ServicesAct(serverID, unit, action string) error {
	return a.services.Act(a.ctx, serverID, unit, action)
}
