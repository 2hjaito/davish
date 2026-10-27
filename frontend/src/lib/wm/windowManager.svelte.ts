import { load, save } from '../stores/persist'
import { clampRect, fitRect, type Rect, type SnapZone, zoneRect } from './geometry'

export interface WindowState {
  id: string
  appId: string
  serverID: string
  title: string
  /** Normal (unsnapped, unmaximized) geometry. */
  rect: Rect
  z: number
  minimized: boolean
  /** 'top' means maximized. */
  snap: SnapZone | null
  /** True briefly after a snap/maximize so geometry changes are animated. */
  animating: boolean
}

export interface OpenOptions {
  title: string
  size: { w: number; h: number }
  /** Preferred position when the app has no saved geometry. */
  at?: { x: number; y: number }
}

/** How an app's window looked when last used, to open it the same way. */
export interface SavedWindow {
  rect: Rect
  snap: SnapZone | null
}

/** One window of a server's layout, as it was left. */
export interface LayoutWindow {
  appId: string
  rect: Rect
  snap: SnapZone | null
  minimized: boolean
}

/** Older versions saved only the rect. */
function toSaved(v: Rect | SavedWindow): SavedWindow {
  return 'rect' in v ? v : { rect: v, snap: null }
}

/** Offset between windows opened one after another, like macOS. */
const CASCADE = 28

let nextId = 1
/** Slightly longer than --dur-slow so the CSS transition can finish. */
const ANIMATION_MS = 320

export class WindowManager {
  windows = $state<WindowState[]>([])
  area = $state<Rect>({ x: 0, y: 30, w: 1280, h: 670 })
  snapPreview = $state<Rect | null>(null)

  /**
   * Geometry by app ("files") and by server and app ("s1/files"): a server
   * opens an app where it last had it, else where the app was last used.
   */
  #saved: Record<string, SavedWindow>
  /** The windows each server had open, back to front. */
  #layouts: Record<string, LayoutWindow[]>
  /** Apps whose windows belong in a layout; the rest need context to reopen. */
  #inLayout: (appId: string) => boolean = () => true
  // Plain records: guards and timer handles are bookkeeping, not UI state.
  #guards: Record<string, () => boolean> = {}
  #animTimers: Record<string, ReturnType<typeof setTimeout>> = {}

  constructor(
    saved: Record<string, Rect | SavedWindow> = {},
    layouts: Record<string, LayoutWindow[]> = {},
  ) {
    this.#saved = Object.fromEntries(Object.entries(saved).map(([k, v]) => [k, toSaved(v)]))
    this.#layouts = layouts
  }

  /** Tells which apps a server's layout keeps (see restoreLayout). */
  keepInLayout(pred: (appId: string) => boolean) {
    this.#inLayout = pred
  }

  /** Whether serverID ever had a layout saved, even an empty one. */
  hasLayout(serverID: string): boolean {
    return serverID in this.#layouts
  }

  /** The windows serverID had open when last seen, back to front. */
  layoutOf(serverID: string): readonly LayoutWindow[] {
    return this.#layouts[serverID] ?? []
  }

  /**
   * Reopens serverID's windows as they were left, unless the workspace
   * already has windows. options gives each app's title and size, or
   * undefined for an app that no longer exists. Returns how many opened.
   */
  restoreLayout(serverID: string, options: (appId: string) => OpenOptions | undefined): number {
    if (this.visible(serverID).length > 0) return 0
    let n = 0
    for (const l of this.layoutOf(serverID)) {
      const o = options(l.appId)
      if (!o || this.isOpen(l.appId, serverID)) continue
      const w = this.open(l.appId, serverID, o)
      w.rect = fitRect(l.rect, this.area)
      w.snap = l.snap
      w.minimized = l.minimized
      n++
    }
    this.#saveLayout(serverID)
    return n
  }

  /** Effective on-screen geometry, accounting for snap and maximize. */
  frame(w: WindowState): Rect {
    return w.snap ? zoneRect(w.snap, this.area) : w.rect
  }

  visible(serverID: string): WindowState[] {
    return this.windows.filter((w) => w.serverID === serverID)
  }

  focused(serverID: string): WindowState | undefined {
    let top: WindowState | undefined
    for (const w of this.windows) {
      if (w.serverID !== serverID || w.minimized) continue
      if (!top || w.z > top.z) top = w
    }
    return top
  }

  isOpen(appId: string, serverID: string): boolean {
    return this.windows.some((w) => w.appId === appId && w.serverID === serverID)
  }

  /** Focuses the app's window on this server, or opens a new one. Returns true if opened. */
  openOrFocus(appId: string, serverID: string, opts: OpenOptions): boolean {
    const existing = this.windows.find((w) => w.appId === appId && w.serverID === serverID)
    if (existing) {
      this.focus(existing.id)
      return false
    }
    this.open(appId, serverID, opts)
    return true
  }

  /**
   * Opens the app's window as it was last left (size, place, maximized or
   * snapped), otherwise at its default size in the middle of the screen.
   * Either way it starts fully on screen and does not cover a window of
   * this workspace at the exact same spot.
   */
  open(appId: string, serverID: string, opts: OpenOptions): WindowState {
    const saved = this.#saved[`${serverID}/${appId}`] ?? this.#saved[appId]
    let rect = fitRect(saved?.rect ?? this.#defaultRect(opts), this.area)
    const taken = (r: Rect) =>
      this.visible(serverID).some((w) => w.rect.x === r.x && w.rect.y === r.y)
    for (let i = 0; i < 10 && taken(rect); i++) {
      rect = fitRect({ ...rect, x: rect.x + CASCADE, y: rect.y + CASCADE }, this.area)
    }
    const win: WindowState = {
      id: `w${nextId++}`,
      appId,
      serverID,
      title: opts.title,
      rect,
      z: this.#topZ() + 1,
      minimized: false,
      snap: saved?.snap ?? null,
      animating: false,
    }
    this.windows.push(win)
    this.#saveLayout(serverID)
    return win
  }

  #defaultRect(opts: OpenOptions): Rect {
    const { w, h } = opts.size
    return {
      x: opts.at?.x ?? Math.round(this.area.x + (this.area.w - w) / 2),
      y: opts.at?.y ?? Math.round(this.area.y + (this.area.h - h) / 2),
      w,
      h,
    }
  }

  /**
   * Closes the window. A guard registered by its app may keep it open (to
   * ask about unsaved work); the app then closes it with force.
   */
  close(id: string, { force = false } = {}) {
    if (!force && this.#guards[id]?.() === false) return
    delete this.#guards[id]
    const w = this.#get(id)
    if (w) this.#remember(w)
    this.windows = this.windows.filter((w) => w.id !== id)
    if (w) this.#saveLayout(w.serverID)
  }

  /** Lets an app veto closing its window: guard returns false to keep it. */
  guardClose(id: string, guard: () => boolean): () => void {
    this.#guards[id] = guard
    return () => {
      if (this.#guards[id] === guard) delete this.#guards[id]
    }
  }

  focus(id: string) {
    const w = this.#get(id)
    if (!w) return
    const changed = w.minimized || w.z !== this.#topZ()
    w.minimized = false
    if (w.z !== this.#topZ()) w.z = this.#topZ() + 1
    if (changed) this.#saveLayout(w.serverID)
  }

  minimize(id: string) {
    const w = this.#get(id)
    if (!w) return
    w.minimized = true
    this.#saveLayout(w.serverID)
  }

  toggleMaximize(id: string) {
    const w = this.#get(id)
    if (w) this.snap(id, w.snap === 'top' ? null : 'top')
  }

  /** Snaps with an animated transition between the old and new frame. */
  snap(id: string, zone: SnapZone | null) {
    const w = this.#get(id)
    if (!w || w.snap === zone) return
    w.snap = zone
    w.animating = true
    this.#remember(w)
    clearTimeout(this.#animTimers[id])
    this.#animTimers[id] = setTimeout(() => {
      const cur = this.#get(id)
      if (cur) cur.animating = false
    }, ANIMATION_MS)
  }

  /** Leaves snap/maximize keeping the current frame, e.g. before a resize. */
  unsnapInPlace(id: string) {
    const w = this.#get(id)
    if (!w?.snap) return
    w.rect = this.frame(w)
    w.snap = null
    w.animating = false
  }

  setRect(id: string, rect: Rect) {
    const w = this.#get(id)
    if (w) w.rect = rect
  }

  /** Persists the window's normal geometry for the next time its app opens. */
  commit(id: string) {
    const w = this.#get(id)
    if (!w) return
    w.rect = clampRect(w.rect, this.area)
    this.#remember(w)
  }

  /**
   * Leaves snap/maximize at the start of a drag, keeping the pointer at the
   * same relative spot on the title bar. Returns the new normal rect.
   */
  unsnapForDrag(id: string, px: number): Rect | undefined {
    const w = this.#get(id)
    if (!w?.snap) return w?.rect
    const f = this.frame(w)
    const ratio = (px - f.x) / f.w
    w.rect = { ...w.rect, x: Math.round(px - w.rect.w * ratio), y: f.y }
    w.snap = null
    w.animating = false
    return w.rect
  }

  #get(id: string) {
    return this.windows.find((w) => w.id === id)
  }

  #topZ() {
    return this.windows.reduce((m, w) => Math.max(m, w.z), 0)
  }

  #remember(w: WindowState) {
    const saved = { rect: { ...w.rect }, snap: w.snap }
    this.#saved[w.appId] = saved
    this.#saved[`${w.serverID}/${w.appId}`] = saved
    save('windows', this.#saved)
    this.#saveLayout(w.serverID)
  }

  #saveLayout(serverID: string) {
    const layout = this.visible(serverID)
      .filter((w) => this.#inLayout(w.appId))
      .sort((a, b) => a.z - b.z)
      .map((w) => ({ appId: w.appId, rect: { ...w.rect }, snap: w.snap, minimized: w.minimized }))
    const before = JSON.stringify(this.#layouts[serverID] ?? [])
    if (JSON.stringify(layout) === before) return
    this.#layouts[serverID] = layout
    save('layouts', this.#layouts)
  }
}

export const wm = new WindowManager(
  load<Record<string, Rect | SavedWindow>>('windows', {}),
  load<Record<string, LayoutWindow[]>>('layouts', {}),
)
