// What a press on a widget is: a click, a drag, or a hold that turns on editing.

/** The pointer must move this far before a press is a drag. */
export const DRAG_START = 5

/** Held this long without becoming a drag, a press turns on widget editing. */
export const HOLD_MS = 500

/** A press on a widget, from pointer down to pointer up. */
export interface Press {
  /** Where the pointer went down. */
  x: number
  y: number
  dragging: boolean
  held: boolean
}

export function startPress(x: number, y: number): Press {
  return { x, y, dragging: false, held: false }
}

/** A drag once the pointer is DRAG_START px from where it went down; a drag stays a drag. */
export function movePress(p: Press, x: number, y: number): Press {
  const dragging = p.dragging || Math.hypot(x - p.x, y - p.y) >= DRAG_START
  return { ...p, dragging }
}

/** The hold timer firing: held, unless the press is already a drag. */
export function holdPress(p: Press): Press {
  return { ...p, held: !p.dragging }
}

/** What the release does: a drag drops, a hold swallows the click, anything else is a click. */
export function endPress(p: Press): 'drop' | 'swallow' | 'click' {
  if (p.dragging) return 'drop'
  return p.held ? 'swallow' : 'click'
}
