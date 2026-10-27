// How one app opens another (docs/ARCHITECTURE.md §5.2), without apps
// importing the registry that imports them.
import { t } from '../i18n/vi'
import { wm } from './windowManager.svelte'

/** Default window sizes the registry shares, for apps opened from here. */
export const TRANSFERS_SIZE = { w: 460, h: 360 }
export const TERMINAL_SIZE = { w: 720, h: 470 }

/** Shows the Transfers window in serverID's workspace. */
export function openTransfers(serverID: string) {
  wm.openOrFocus('transfers', serverID, { title: t.apps.transfers, size: TRANSFERS_SIZE })
}

interface TerminalRequest {
  serverID: string
  dir: string
}

/** Folders waiting for a terminal tab; the Terminal app takes them. */
class TerminalRequests {
  list = $state<TerminalRequest[]>([])

  /** Removes and returns the folders asked for on serverID. */
  take(serverID: string): string[] {
    const mine = this.list.filter((r) => r.serverID === serverID)
    if (mine.length > 0) this.list = this.list.filter((r) => r.serverID !== serverID)
    return mine.map((r) => r.dir)
  }
}

export const terminalRequests = new TerminalRequests()

/** Opens a terminal tab in dir, in serverID's Terminal window. */
export function openTerminalAt(serverID: string, dir: string) {
  terminalRequests.list.push({ serverID, dir })
  wm.openOrFocus('terminal', serverID, { title: t.apps.terminal, size: TERMINAL_SIZE })
}
