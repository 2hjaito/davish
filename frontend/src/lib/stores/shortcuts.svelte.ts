import type { ShortcutKind } from '../api/shortcuts'
import { type Cell, freeCell, type Grid } from '../shell/iconGrid'
import type { IconName } from '../ui/icons'
import { load, save } from './persist'

/** An icon on one server's desktop. Holds a path, never the file's content. */
export interface Shortcut extends Cell {
  path: string
  name: string
  kind: ShortcutKind
  /** The user's name for it. Only Davish sees it: the file keeps its name. */
  label?: string
  icon?: IconName
  color?: string
  /** Id of a picture from this computer, which wins over icon and color. */
  image?: string
  /** SHA-256 of the script allowed to run without asking (docs/adr/0013). */
  trusted?: string
}

/** What the customize dialog changes; image is the picture itself. */
export type ShortcutLook = Pick<Shortcut, 'label' | 'icon' | 'color'> & { image: string }

const KEY = 'desktop-shortcuts'

// A picture gets a settings key of its own: one key holds 64 KiB, and an icon
// moving around should not write every picture again.
const imageKey = (id: string) => `desktop-shortcut-image:${id}`
const IMAGE_PREFIX = 'data:image/png;base64,'

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

  /** The picture of s, or nothing. Only an image we encoded may reach an <img>. */
  image(s: Pick<Shortcut, 'image'>): string {
    if (!s.image) return ''
    const data = load<unknown>(imageKey(s.image), '')
    return typeof data === 'string' && data.startsWith(IMAGE_PREFIX) ? data : ''
  }

  /** A new picture gets a new id, so whoever draws the icon sees the change. */
  customize(serverID: string, path: string, { image, ...look }: ShortcutLook) {
    const item = this.find(serverID, path)
    if (!item) return
    let id = item.image
    if (image !== this.image(item)) {
      this.#dropImage(item)
      id = image ? `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}` : undefined
      if (id) save(imageKey(id), image)
    }
    this.#update(serverID, path, { ...look, image: id })
  }

  remove(serverID: string, path: string) {
    const item = this.find(serverID, path)
    if (item) this.#dropImage(item)
    this.byServer[serverID] = this.of(serverID).filter((s) => s.path !== path)
    save(KEY, this.byServer)
  }

  forgetServer(serverID: string) {
    if (!(serverID in this.byServer)) return
    this.of(serverID).forEach((s) => this.#dropImage(s))
    delete this.byServer[serverID]
    save(KEY, this.byServer)
  }

  // Settings have no delete; null reads as nothing and frees the picture.
  #dropImage(s: Shortcut) {
    if (s.image) save(imageKey(s.image), null)
  }

  #update(serverID: string, path: string, change: Partial<Shortcut>) {
    this.byServer[serverID] = this.of(serverID).map((s) =>
      s.path === path ? { ...s, ...change } : s,
    )
    save(KEY, this.byServer)
  }
}

export const shortcuts = new Shortcuts()
