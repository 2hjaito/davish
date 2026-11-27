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

/**
 * Where each of `count` tiles is centered, in px from the dock's center, with
 * the tiles at `scales` (missing ones at 1). As in Dock.svelte's .slot, a tile
 * of size `tile` grows by its margins, pushing its neighbours aside, and the
 * dock stays centered while it grows.
 */
export function slotCenters(
  count: number,
  scales: readonly number[],
  step: number,
  tile: number,
): number[] {
  const extra = Array.from({ length: count }, (_, i) => ((scales[i] ?? 1) - 1) * tile)
  const total = extra.reduce((a, b) => a + b, 0)
  let before = 0
  return extra.map((e, i) => {
    const center = slotCenter(i, count, step) + before + e / 2 - total / 2
    before += e
    return center
  })
}

/** The slot from min to max whose center (in ascending `centers`) is nearest x. */
export function nearestSlot(
  centers: readonly number[],
  x: number,
  min: number,
  max: number,
): number {
  let i = min
  while (i < max && x > ((centers[i] ?? 0) + (centers[i + 1] ?? 0)) / 2) i++
  return i
}

/**
 * Frame-rate independent exponential approach from `current` toward `target`,
 * landing on it once within `settle` (a scale by default; pass ~0.5 for px).
 */
export function approach(current: number, target: number, dtMs: number, settle = 0.0005): number {
  const next = target + (current - target) * Math.exp(-dtMs / SMOOTH_MS)
  return Math.abs(next - target) < settle ? target : next
}
