// How one app opens another (docs/ARCHITECTURE.md §5.2), without apps
// importing the registry that imports them.
import { t } from '../i18n/vi'
import { wm } from './windowManager.svelte'

/** Size of the Transfers window; the registry uses it too. */
export const TRANSFERS_SIZE = { w: 460, h: 360 }

/** Shows the Transfers window in serverID's workspace. */
export function openTransfers(serverID: string) {
  wm.openOrFocus('transfers', serverID, { title: t.apps.transfers, size: TRANSFERS_SIZE })
}
