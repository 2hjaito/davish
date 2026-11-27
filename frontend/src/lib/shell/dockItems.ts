// What the dock shows: the pinned apps in the user's order, then apps
// with a window open that are not pinned, so a minimized window of an
// app opened from Launchpad can always be brought back.

/**
 * The apps pinned on a first run: the everyday ones. The rest are in
 * Launchpad and Spotlight, and can be pinned from there.
 */
export const DEFAULT_PINS: readonly string[] = [
  'terminal',
  'files',
  'monitor',
  'services',
  'docker',
  'logs',
  'editor',
  'settings',
]

/** The apps to show, pinned first; ids no longer in all are skipped. */
export function dockItems<T extends { id: string }>(
  all: readonly T[],
  pinned: readonly string[],
  isOpen: (id: string) => boolean,
): T[] {
  const shown = pinned
    .map((id) => all.find((a) => a.id === id))
    .filter((a): a is T => a !== undefined)
  const running = all.filter((a) => !pinned.includes(a.id) && isOpen(a.id))
  return [...shown, ...running]
}

/** pinned with id added at the end, or removed. */
export function togglePin(pinned: readonly string[], id: string): string[] {
  return pinned.includes(id) ? pinned.filter((p) => p !== id) : [...pinned, id]
}

/**
 * pinned with id moved to just before `before`, or to the end when
 * before is null or not pinned; id is pinned if it was not. Pins the
 * dock is not showing (apps another server's system has) keep their places.
 */
export function movePin(pinned: readonly string[], id: string, before: string | null): string[] {
  const rest = pinned.filter((p) => p !== id)
  const at = before === null ? -1 : rest.indexOf(before)
  return at < 0 ? [...rest, id] : [...rest.slice(0, at), id, ...rest.slice(at)]
}

/** list with the item at `from` moved to index `to`. */
export function moveItem<T>(list: readonly T[], from: number, to: number): T[] {
  const out = [...list]
  out.splice(to, 0, ...out.splice(from, 1))
  return out
}
