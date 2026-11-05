import type { ShortcutKind } from '../api/shortcuts'
import { type Cell, freeCell, type Grid } from '../shell/iconGrid'
import { load, save } from './persist'

/** An icon on one server's desktop. Holds a path, never the file's content. */
export interface Shortcut extends Cell {
  path: string
  name: string
  kind: ShortcutKind
  /** SHA-256 of the script allowed to run without asking (docs/adr/0013). */
  trusted?: string
}

const KEY = 'desktop-shortcuts'

class Shortcuts {
  /** Icons per server: a shortcut never shows on another server's desktop. */
  byServer = $state<Record<string, Shortcut[]>>(load(KEY, {}))

  of(serverID: string): Shortcut[] {
    return this.byServer[serverID] ?? []
  }

  find(serverID: string, path: string): Shortcut | undefined {
    return this.of(serverID).find((s) => s.path === path)
  }

  add(serverID: string, s: Omit<Shortcut, 'col' | 'row'>, want: Cell, grid: Grid) {
    const cell = freeCell(want, this.of(serverID), grid)
    this.byServer[serverID] = [...this.of(serverID), { ...s, ...cell }]
    save(KEY, this.byServer)
  }

  /** Moves an icon to the free cell nearest want. */
  move(serverID: string, path: string, want: Cell, grid: Grid) {
    const others = this.of(serverID).filter((s) => s.path !== path)
    this.#update(serverID, path, freeCell(want, others, grid))
  }

  /** Lets the script with this hash run without asking; no hash takes that back. */
  trust(serverID: string, path: string, hash: string | undefined) {
    this.#update(serverID, path, { trusted: hash })
  }

  setKind(serverID: string, path: string, kind: ShortcutKind) {
    this.#update(serverID, path, { kind })
  }

  remove(serverID: string, path: string) {
    this.byServer[serverID] = this.of(serverID).filter((s) => s.path !== path)
    save(KEY, this.byServer)
  }

  forgetServer(serverID: string) {
    if (!(serverID in this.byServer)) return
    delete this.byServer[serverID]
    save(KEY, this.byServer)
  }

  #update(serverID: string, path: string, change: Partial<Shortcut>) {
    this.byServer[serverID] = this.of(serverID).map((s) =>
      s.path === path ? { ...s, ...change } : s,
    )
    save(KEY, this.byServer)
  }
}

export const shortcuts = new Shortcuts()
