// Pure helpers of the Security overview: hour labels and bar lengths.

/** "2026-10-28T15" as "10-28 15h", the server's clock. */
export function hourLabel(hour: string): string {
  const m = /^\d{4}-(\d{2})-(\d{2})T(\d{2})$/.exec(hour)
  return m ? `${m[1]}-${m[2]} ${m[3]}h` : hour
}

/** Each count as a share of the largest, in percent; a count above 0
 *  keeps a sliver so it never reads as none. */
export function shares(counts: readonly number[]): number[] {
  const top = Math.max(0, ...counts)
  return counts.map((n) => (top === 0 || n <= 0 ? 0 : Math.max(2, (n / top) * 100)))
}
