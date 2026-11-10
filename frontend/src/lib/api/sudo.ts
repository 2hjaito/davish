// Typed wrappers for the sudo bindings (internal/app/sudo.go, ADR 0018).
import * as App from '../../../wailsjs/go/app/App'
import { hasRuntime } from './runtime'

/** How a server's user reaches root. */
export type SudoState = 'root' | 'nopassword' | 'password' | 'ready' | 'none'

export const sudoState = (serverID: string): Promise<SudoState> =>
  hasRuntime ? (App.SudoState(serverID) as Promise<SudoState>) : Promise.resolve('password')

/** Checks the password with sudo and keeps it in memory for this session. */
export const sudoSetPassword = (serverID: string, password: string): Promise<void> =>
  hasRuntime
    ? App.SudoSetPassword(serverID, password)
    : password === 'wrong'
      ? Promise.reject({ code: 'SUDO_PASSWORD_WRONG', message: 'sudo did not accept the password' })
      : Promise.resolve()

/** Wipes the server's sudo password. */
export const sudoForget = (serverID: string): Promise<void> =>
  hasRuntime ? App.SudoForget(serverID) : Promise.resolve()
