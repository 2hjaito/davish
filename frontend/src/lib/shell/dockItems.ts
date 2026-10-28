// What the dock shows: the pinned apps in the user's order, then apps
// with a window open that are not pinned, so a minimized window of an
// app opened from Launchpad can always be brought back.

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
