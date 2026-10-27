// What the Files app remembers between folders and between runs.
import { load, save } from '../../lib/stores/persist'
import { joinPath, within } from './listing'

export interface FolderView {
  scrollTop: number
  selected: string | null
}

/**
 * The child of to that leads back down to from, when going up: Finder
 * selects the folder you just came out of. Null when to is not above from.
 */
export function cameFrom(from: string, to: string): string | null {
  if (from === to || !within(from, to)) return null
  const rest = from.slice(to === '/' ? 1 : to.length + 1)
  const name = rest.split('/')[0]
  return name ? joinPath(to, name) : null
}

// Its own key, read fresh each time: several Files windows (one per
// server) share it and must not overwrite each other's entries.
const LAST_DIR = 'files.lastDir'

export function lastDir(serverID: string): string | null {
  return load<Record<string, string>>(LAST_DIR, {})[serverID] ?? null
}

export function rememberDir(serverID: string, path: string) {
  const all = load<Record<string, string>>(LAST_DIR, {})
  if (all[serverID] === path) return
  save(LAST_DIR, { ...all, [serverID]: path })
}
