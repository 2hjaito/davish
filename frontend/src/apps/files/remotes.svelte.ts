// Where the other servers are, for the "other servers" part of the Files
// sidebar: their home folders, asked once per server and session.
import { filesHome } from '../../lib/api/files'
import { hasRuntime } from '../../lib/api/runtime'
import { mockHome } from './mockFs'

/** A folder of another server that a dragged file can be copied into. */
export interface DropTarget {
  serverID: string
  path: string
}

class Homes {
  byServer = $state<Record<string, string>>({})
  #asked = new Set<string>()

  /** Learns serverID's home folder, once; serverID must be connected. */
  ensure(serverID: string) {
    if (this.#asked.has(serverID)) return
    this.#asked.add(serverID)
    const ask = hasRuntime ? filesHome(serverID) : Promise.resolve(mockHome)
    ask.then(
      (home) => (this.byServer[serverID] = home),
      () => this.#asked.delete(serverID),
    )
  }
}

export const homes = new Homes()

/** The drop target under (x, y): a sidebar entry of another server. */
export function dropTargetAt(x: number, y: number): DropTarget | null {
  if (!Number.isFinite(x) || !Number.isFinite(y)) return null
  for (const el of document.elementsFromPoint(x, y)) {
    const t = el.closest<HTMLElement>('[data-drop-server]')
    if (t && !(t as HTMLButtonElement).disabled) {
      const serverID = t.dataset.dropServer
      const path = t.dataset.dropPath
      if (serverID && path) return { serverID, path }
    }
  }
  return null
}
