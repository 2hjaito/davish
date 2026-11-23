// The numbers behind the widgets' drawings: a ring filled to a share and a
// grid of dots filled in parts. Pure, so they can be tested.

const clamp01 = (v: number) => (Number.isFinite(v) ? Math.min(1, Math.max(0, v)) : 0)

/**
 * The stroke-dasharray that fills a ring of radius r to value (0 to 1): the
 * filled arc, then a gap as long as the rest.
 */
export function ringDash(value: number, r: number): string {
  const c = 2 * Math.PI * r
  const filled = clamp01(value) * c
  return `${filled.toFixed(2)} ${(c - filled + 0.01).toFixed(2)}`
}

/**
 * How many of n cells a share fills: rounded, but never none for a share that
 * is not zero, and never all for one that is not whole, so a small or a near
 * full reading still shows.
 */
export function cellsFor(share: number, n: number): number {
  const v = clamp01(share)
  if (v === 0 || n <= 0) return 0
  if (v === 1) return n
  return Math.min(n - 1, Math.max(1, Math.round(v * n)))
}

export interface Part<K extends string> {
  kind: K
  count: number
}

/** n cells: each part's count in order, then 'empty' for the rest; cut at n. */
export function dotCells<K extends string>(n: number, parts: readonly Part<K>[]): (K | 'empty')[] {
  const out: (K | 'empty')[] = []
  for (const p of parts) for (let i = 0; i < p.count && out.length < n; i++) out.push(p.kind)
  while (out.length < n) out.push('empty')
  return out.slice(0, Math.max(0, n))
}

/**
 * A size as formatBytes writes it ("42,4 GB") in its number and its unit, so
 * the number can be set large and the unit small. Without a space it is all number.
 */
export function splitUnit(text: string): { value: string; unit: string } {
  const i = text.lastIndexOf(' ')
  return i < 0 ? { value: text, unit: '' } : { value: text.slice(0, i), unit: text.slice(i + 1) }
}
