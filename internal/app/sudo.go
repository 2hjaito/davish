package app

import (
	"time"

	"davish/internal/apperr"
)

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

// SudoForgetAfter sets how many minutes an unused sudo password is kept;
// 0 keeps it until the session ends (ADR 0018).
func (a *App) SudoForgetAfter(minutes int) error {
	if minutes < 0 || minutes > 24*60 {
		return apperr.New(apperr.InvalidInput, "minutes out of range")
	}
	a.sudo.SetForgetAfter(time.Duration(minutes) * time.Minute)
	return nil
}
