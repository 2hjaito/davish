// Geometry of the drag menu (docs/DESIGN.md, "Thao tác bằng kéo thả"):
// petals around the pointer, clockwise from the top, and an outer ring
// that opens beyond a petal with choices of its own.

/** Radii in px: center hole, petals, outer ring. */
export const RADII = { inner: 34, outer: 104, ring: 164 } as const

export interface Hit {
  /** Petal under the pointer, or null in the center or outside. */
  petal: number | null
  /** Choice in the outer ring beyond that petal, if any. */
  sub: number | null
}

export const NO_HIT: Hit = { petal: null, sub: null }

/** Degrees clockwise from straight up, 0 to 360. */
export function bearing(dx: number, dy: number): number {
  const deg = (Math.atan2(dy, dx) * 180) / Math.PI + 90
  return (deg + 360) % 360
}

/** The petal's span in degrees, centered on its direction. */
export function petalSpan(i: number, count: number): [number, number] {
  const span = 360 / count
  return [i * span - span / 2, i * span + span / 2]
}

/**
 * What lies under the pointer at (dx, dy) from the center. subCount(i) is
 * how many outer-ring choices petal i has; the ring splits the petal's
 * span between them.
 */
export function hitTest(
  dx: number,
  dy: number,
  count: number,
  subCount: (i: number) => number,
): Hit {
  const r = Math.hypot(dx, dy)
  if (count === 0 || r < RADII.inner || r > RADII.ring) return NO_HIT
  const span = 360 / count
  // Shift by half a petal so petal 0 is centered on the top.
  const a = (bearing(dx, dy) + span / 2) % 360
  const petal = Math.min(count - 1, Math.floor(a / span))
  if (r <= RADII.outer) return { petal, sub: null }
  const n = subCount(petal)
  if (n === 0) return NO_HIT
  const within = (a - petal * span) / span
  return { petal, sub: Math.min(n - 1, Math.floor(within * n)) }
}

/** Point at radius r and bearing deg from (cx, cy). */
export function polar(cx: number, cy: number, r: number, deg: number): { x: number; y: number } {
  const rad = ((deg - 90) * Math.PI) / 180
  return { x: cx + r * Math.cos(rad), y: cy + r * Math.sin(rad) }
}

/** SVG path of the ring sector between radii r0, r1 and bearings a0, a1. */
export function sectorPath(
  cx: number,
  cy: number,
  r0: number,
  r1: number,
  a0: number,
  a1: number,
): string {
  const large = a1 - a0 > 180 ? 1 : 0
  const p0 = polar(cx, cy, r1, a0)
  const p1 = polar(cx, cy, r1, a1)
  const p2 = polar(cx, cy, r0, a1)
  const p3 = polar(cx, cy, r0, a0)
  const f = (n: number) => n.toFixed(2)
  return [
    `M ${f(p0.x)} ${f(p0.y)}`,
    `A ${r1} ${r1} 0 ${large} 1 ${f(p1.x)} ${f(p1.y)}`,
    `L ${f(p2.x)} ${f(p2.y)}`,
    `A ${r0} ${r0} 0 ${large} 0 ${f(p3.x)} ${f(p3.y)}`,
    'Z',
  ].join(' ')
}
