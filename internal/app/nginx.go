package app

import "davish/internal/nginx"

// NginxList returns the sites of sites-available, sites-enabled and
// conf.d, with the certificates of enabled sites.
func (a *App) NginxList(serverID string) ([]nginx.Site, error) {
	return a.nginx.List(a.ctx, serverID)
}

// NginxPlanToggle returns the exact command that turns a site on or off.
func (a *App) NginxPlanToggle(serverID, name string, enable bool) (nginx.TogglePlan, error) {
	return a.nginx.PlanToggle(a.ctx, serverID, name, enable)
}

// NginxToggle runs the confirmed command from NginxPlanToggle.
func (a *App) NginxToggle(serverID, name string, enable bool, command string) error {
	return a.nginx.Toggle(a.ctx, serverID, name, enable, command)
}

// NginxTest runs nginx -t.
func (a *App) NginxTest(serverID string) (nginx.TestResult, error) {
	return a.nginx.Test(a.ctx, serverID)
}

// NginxReloadCommand is what NginxReload runs, for the confirmation.
func (a *App) NginxReloadCommand() string {
	return nginx.ReloadCommand
}

// NginxReload tests the configuration and reloads nginx if it passes.
func (a *App) NginxReload(serverID string) (nginx.ReloadResult, error) {
	return a.nginx.Reload(a.ctx, serverID)
}

// NginxCheckCerts notifies TLS certificates of enabled sites that expire
// within 14 days; nothing happens on a server without nginx.
func (a *App) NginxCheckCerts(serverID string) error {
	return a.nginx.CheckCerts(a.ctx, serverID)
}
