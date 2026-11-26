/** Scale of a tile right under the pointer. */
export const MAX_SCALE = 1.2
/** Upward lift, in px, of a tile at MAX_SCALE; smaller tiles lift proportionally. */
export const MAX_LIFT = 8
/** How far the effect reaches, in tile steps (tile + gap) from the pointer. */
const REACH = 2
/** Smoothing time constant, in ms: higher feels heavier, lower snappier. */
const SMOOTH_MS = 70

/**
 * Per-tile scales for a pointer at `x` px from the dock's center. Distances use
 * the tiles' resting centers, so the magnified layout never feeds back into the
 * input. The falloff is a raised cosine: smooth at the peak and at the edge.
 */
export function magnify(x: number, count: number, step: number): number[] {
  const reach = REACH * step
  return Array.from({ length: count }, (_, i) => {
    const d = Math.abs(x - slotCenter(i, count, step))
    if (d >= reach) return 1
    return 1 + ((MAX_SCALE - 1) * (Math.cos((Math.PI * d) / reach) + 1)) / 2
  })
}

/** Resting center of slot `i` of `count`, in px from the dock's center. */
export function slotCenter(i: number, count: number, step: number): number {
  return (i - (count - 1) / 2) * step
}

/** The slot whose resting center is nearest `x` px from the dock's center. */
export function slotAt(x: number, count: number, step: number): number {
  return step > 0 ? Math.round(x / step + (count - 1) / 2) : 0
}

/** Frame-rate independent exponential approach from `current` toward `target`. */
export function approach(current: number, target: number, dtMs: number): number {
  const next = target + (current - target) * Math.exp(-dtMs / SMOOTH_MS)
  return Math.abs(next - target) < 0.0005 ? target : next
}
