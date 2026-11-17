// A list beside its detail, with the line between them dragged by the user.

/** The narrowest the list and the detail may get, in px. */
export const SPLIT_MIN = { list: 220, detail: 280 } as const

/**
 * The share of total the list takes when its edge is at x px, kept so both
 * sides stay usable. Null when total is too small to split.
 */
export function splitRatio(x: number, total: number): number | null {
  if (!Number.isFinite(x) || !(total >= SPLIT_MIN.list + SPLIT_MIN.detail)) return null
  const px = Math.min(total - SPLIT_MIN.detail, Math.max(SPLIT_MIN.list, x))
  return px / total
}

/** A stored ratio, or null when it is missing or junk. */
export function storedRatio(v: unknown): number | null {
  return typeof v === 'number' && v > 0.05 && v < 0.95 ? v : null
}
