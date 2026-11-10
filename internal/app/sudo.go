package app

// SudoState tells how the server's user reaches root: root, nopassword,
// password, ready or none. It never asks for a password.
func (a *App) SudoState(serverID string) (string, error) {
	return a.sudo.State(a.ctx, serverID)
}

// SudoSetPassword checks the sudo password and keeps it in memory for this
// session only (ADR 0018).
func (a *App) SudoSetPassword(serverID, password string) error {
	if err := a.lock.Check(); err != nil {
		return err
	}
	return a.sudo.SetPassword(a.ctx, serverID, []byte(password))
}

// SudoForget wipes the server's sudo password.
func (a *App) SudoForget(serverID string) {
	a.sudo.Forget(serverID)
}
