// What opening and adding a desktop icon decide, apart from drawing it.
import { hasRuntime } from '../api/runtime'
import { shortcutInspect, type ShortcutTarget } from '../api/shortcuts'
import { deploys } from '../stores/deploys.svelte'
import { type Shortcut, shortcuts } from '../stores/shortcuts.svelte'
import { wm } from '../wm/windowManager.svelte'
import { cellAt, gridOf } from './iconGrid'

/** Is the point on bare wallpaper, not under a window or the dock? */
export function onSurface(x: number, y: number): boolean {
  return !!document.elementFromPoint(x, y)?.closest('[data-desktop-surface]')
}

/** Puts path on serverID's desktop; false when it is there already. */
export async function addShortcut(
  serverID: string,
  path: string,
  at?: { x: number; y: number },
): Promise<boolean> {
  if (shortcuts.find(serverID, path)) return false
  const { name, kind } = await shortcutInspect(serverID, path)
  const grid = gridOf(wm.area)
  const want = at ? cellAt(at.x, at.y, wm.area, grid) : { col: 0, row: 0 }
  shortcuts.add(serverID, { path, name, kind }, want, grid)
  return true
}

/** Must the user see this script before it runs? */
export function needsConfirm(s: Pick<Shortcut, 'trusted'>, target: ShortcutTarget): boolean {
  return !target.hash || target.hash !== s.trusted
}

/** The script was allowed to run directly, but is not that script any more. */
export function hasChanged(s: Pick<Shortcut, 'trusted'>, target: ShortcutTarget): boolean {
  return !!s.trusted && target.hash !== s.trusted
}

const PREVIEW_RUN_MS = 12_000

/** The browser preview has no server: marks arrive as a real run's would. */
export function simulateRun(nonce: string) {
  if (hasRuntime) return
  deploys.apply({ nonce, kind: 'begin', steps: 1 })
  deploys.apply({ nonce, kind: 'step', step: 1 })
  setTimeout(() => deploys.apply({ nonce, kind: 'end', exit: 0 }), PREVIEW_RUN_MS)
}
