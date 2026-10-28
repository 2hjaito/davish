package app

import "davish/internal/security"

// SecurityReport reads SSH logins, failed attempts, ufw and fail2ban.
func (a *App) SecurityReport(serverID string) (security.Report, error) {
	return a.security.Report(a.ctx, serverID)
}
