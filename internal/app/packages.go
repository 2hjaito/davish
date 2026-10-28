package app

import "davish/internal/packages"

// PackagesInventory lists the installed packages and the pending updates.
func (a *App) PackagesInventory(serverID string) (packages.Inventory, error) {
	return a.packages.Inventory(a.ctx, serverID)
}

// PackagesApps finds the runtimes and server programs installed.
func (a *App) PackagesApps(serverID string) ([]packages.App, error) {
	return a.packages.Apps(a.ctx, serverID)
}

// PackagesNpmGlobals lists the packages installed with npm -g.
func (a *App) PackagesNpmGlobals(serverID string) ([]packages.NpmGlobal, error) {
	return a.packages.NpmGlobals(a.ctx, serverID)
}

// PackagesPlan is the sudo command for an action on packages, to type into
// a terminal; Davish never runs it.
func (a *App) PackagesPlan(manager, op string, names []string) (string, error) {
	return packages.PlanAction(manager, op, names)
}

// PackagesCheckSecurity notifies pending security fixes, on connection.
func (a *App) PackagesCheckSecurity(serverID string) error {
	return a.packages.CheckSecurity(a.ctx, serverID)
}
