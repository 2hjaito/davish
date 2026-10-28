package app

import "davish/internal/lock"

// LockState tells whether a lock password is set, whether the app is
// locked, and whether Touch ID can open it.
func (a *App) LockState() lock.State {
	return a.lock.State()
}

// LockNow locks the app (⌘L, or idle for too long).
func (a *App) LockNow() {
	a.lock.Lock()
}

// Unlock opens the app with the lock password.
func (a *App) Unlock(password string) error {
	return a.lock.Unlock(password)
}

// UnlockTouchID opens the app with Touch ID; reason is shown on the sheet.
func (a *App) UnlockTouchID(reason string) error {
	return a.lock.UnlockBiometric(a.ctx, reason)
}

// LockSetPassword sets the lock password, or changes it given the current one.
func (a *App) LockSetPassword(current, next string) error {
	return a.lock.SetPassword(current, next)
}

// LockDisable turns the lock off given the current password.
func (a *App) LockDisable(current string) error {
	return a.lock.Disable(current)
}

// LockReset is "forgot the password": it removes every saved server secret
// and the lock password, then unlocks.
func (a *App) LockReset() error {
	return a.lock.Reset()
}
