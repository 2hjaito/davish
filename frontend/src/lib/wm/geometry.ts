export interface Rect {
  x: number
  y: number
  w: number
  h: number
}

export type Edge = 'n' | 's' | 'e' | 'w' | 'ne' | 'nw' | 'se' | 'sw'
export type SnapZone = 'left' | 'right' | 'top'

export const MIN_W = 320
export const MIN_H = 200
/** Pixels from the work-area edge where dragging triggers a snap. */
export const SNAP_THRESHOLD = 6
/** Minimum part of a window that must stay on screen. */
const KEEP_VISIBLE = 80
const TITLEBAR_H = 38

/** Resizes rect from the given edge by (dx, dy), honoring the minimum size. */
export function resizeRect(r: Rect, edge: Edge, dx: number, dy: number): Rect {
  let { x, y, w, h } = r
  if (edge.includes('e')) w = Math.max(MIN_W, r.w + dx)
  if (edge.includes('s')) h = Math.max(MIN_H, r.h + dy)
  if (edge.includes('w')) {
    w = Math.max(MIN_W, r.w - dx)
    x = r.x + r.w - w
  }
  if (edge.includes('n')) {
    h = Math.max(MIN_H, r.h - dy)
    y = r.y + r.h - h
  }
  return { x, y, w, h }
}

/** Keeps the title bar reachable inside the work area. */
export function clampRect(r: Rect, area: Rect): Rect {
  const w = Math.min(r.w, area.w)
  const h = Math.min(r.h, area.h)
  const x = Math.min(Math.max(r.x, area.x - w + KEEP_VISIBLE), area.x + area.w - KEEP_VISIBLE)
  const y = Math.min(Math.max(r.y, area.y), area.y + area.h - TITLEBAR_H)
  return { x, y, w, h }
}

/** Returns the snap zone under the pointer, if any. */
export function snapZone(px: number, py: number, area: Rect): SnapZone | null {
  if (py <= area.y + SNAP_THRESHOLD) return 'top'
  if (px <= area.x + SNAP_THRESHOLD) return 'left'
  if (px >= area.x + area.w - SNAP_THRESHOLD) return 'right'
  return null
}

export function zoneRect(zone: SnapZone, area: Rect): Rect {
  const half = Math.round(area.w / 2)
  switch (zone) {
    case 'top':
      return { ...area }
    case 'left':
      return { x: area.x, y: area.y, w: half, h: area.h }
    case 'right':
      return { x: area.x + half, y: area.y, w: area.w - half, h: area.h }
  }
}
