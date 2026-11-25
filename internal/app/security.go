package app

import (
	"davish/internal/security"
	"davish/internal/store"
)

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
	return a.security.Apply(a.change(), serverID, action, ip, command)
}

// SecurityPasswordStatus is what sshd does about password logins and
// whether turning them off is safe to offer.
func (a *App) SecurityPasswordStatus(serverID string) (security.PasswordStatus, error) {
	srv, err := a.store.Server(a.ctx, serverID)
	if err != nil {
		return security.PasswordStatus{}, err
	}
	return a.security.ReadPasswordStatus(a.ctx, serverID, srv.Auth != store.AuthPassword, srv.ControlPath != "")
}

// SecurityPasswordPlan is the commands that turn password logins off and
// back on, to show before they run.
func (a *App) SecurityPasswordPlan(root bool) security.PasswordPlan {
	return security.PlanPassword(root)
}

// SecurityDisablePassword runs the confirmed command that turns password
// logins off, with the key checked before and after (ADR 0029).
func (a *App) SecurityDisablePassword(serverID, command string) error {
	if err := a.lock.Check(); err != nil {
		return err
	}
	return a.security.DisablePassword(a.change(), serverID, command)
}

// SecurityEnablePassword runs the confirmed command that undoes it.
func (a *App) SecurityEnablePassword(serverID, command string) error {
	if err := a.lock.Check(); err != nil {
		return err
	}
	return a.security.EnablePassword(a.change(), serverID, command)
}
