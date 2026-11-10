package app

import "davish/internal/security"

// SecurityReport reads SSH logins, failed attempts, ufw and fail2ban.
func (a *App) SecurityReport(serverID string) (security.Report, error) {
	return a.security.Report(a.ctx, serverID)
}

// SecurityPlan is the command that blocks, unblocks, bans or unbans ip,
// to show before it runs.
func (a *App) SecurityPlan(action, ip string, root bool) (security.Plan, error) {
	return security.PlanAction(action, ip, root)
}

// SecurityApply runs the confirmed command of SecurityPlan.
func (a *App) SecurityApply(serverID, action, ip, command string) error {
	if err := a.lock.Check(); err != nil {
		return err
	}
	return a.security.Apply(a.ctx, serverID, action, ip, command)
}
