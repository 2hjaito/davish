// Typed wrappers for the app lock bindings (internal/app/lock.go).
import * as App from '../../../wailsjs/go/app/App'

/** The lock:state payload (internal/lock.State). */
export interface LockState {
  /** A lock password is set. */
  enabled: boolean
  locked: boolean
  /** Touch ID can unlock. */
  biometric: boolean
  /** When a password may be tried again, ms since epoch; 0 means now. */
  retryAt: number
}

export const lockState = (): Promise<LockState> => App.LockState()
export const lockNow = (): Promise<void> => App.LockNow()
export const unlock = (password: string): Promise<void> => App.Unlock(password)
/** reason is shown on the Touch ID sheet. */
export const unlockTouchID = (reason: string): Promise<void> => App.UnlockTouchID(reason)
export const lockSetPassword = (current: string, next: string): Promise<void> =>
  App.LockSetPassword(current, next)
export const lockDisable = (current: string): Promise<void> => App.LockDisable(current)
/** Removes every saved server secret and the lock password, then unlocks. */
export const lockReset = (): Promise<void> => App.LockReset()
