// Pure helpers for the charts, kept apart from the components so they can be
// unit tested.

export interface ChartSeries<S> {
  label: string
  /** A CSS color, normally a --chart-N token. */
  color: string
  value: (s: S) => number
}

export interface Segment {
  label: string
  value: number
  color: string
}

/**
 * Rounds a maximum up to a clean axis value: 1, 2 or 5 times a power of ten,
 * or with binary=true times a power of 1024 so byte axes read "2 MB", not
 * "1,9 MB". Never below floor, so an idle chart is not all noise.
 */
export function niceMax(v: number, { binary = false, floor = 1 } = {}): number {
  const target = Math.max(v, floor)
  if (!Number.isFinite(target) || target <= 0) return floor
  let unit = 1
  if (binary) {
    while (target / unit >= 1024) unit *= 1024
  }
  const scaled = target / unit
  const pow = 10 ** Math.floor(Math.log10(scaled))
  const step = [1, 2, 5, 10].find((m) => m * pow >= scaled - 1e-9) ?? 10
  return step * pow * unit
}

export interface Point {
  x: number
  y: number
}

/**
 * SVG path data for a line through points, starting a new segment wherever
 * two samples are further apart than maxGap (the server was unreachable):
 * the chart shows a hole instead of a made-up straight line.
 */
export function linePath(points: Point[], times: number[], maxGap: number): string {
  let d = ''
  for (let i = 0; i < points.length; i++) {
    const p = points[i]
    if (!p) continue
    const gap = i === 0 || (times[i] ?? 0) - (times[i - 1] ?? 0) > maxGap
    d += `${gap ? 'M' : 'L'}${p.x.toFixed(1)} ${p.y.toFixed(1)}`
  }
  return d
}

/** Like linePath, but each segment is closed down to baseline as an area. */
export function areaPath(
  points: Point[],
  times: number[],
  maxGap: number,
  baseline: number,
): string {
  let d = ''
  let start: Point | undefined
  let last: Point | undefined
  const close = () => {
    if (start && last) d += `L${last.x.toFixed(1)} ${baseline}L${start.x.toFixed(1)} ${baseline}Z`
  }
  for (let i = 0; i < points.length; i++) {
    const p = points[i]
    if (!p) continue
    if (i === 0 || (times[i] ?? 0) - (times[i - 1] ?? 0) > maxGap) {
      close()
      start = p
      d += `M${p.x.toFixed(1)} ${p.y.toFixed(1)}`
    } else {
      d += `L${p.x.toFixed(1)} ${p.y.toFixed(1)}`
    }
    last = p
  }
  close()
  return d
}

/** Index of the time nearest to t; times are sorted. -1 when empty. */
export function nearestIndex(times: number[], t: number): number {
  if (times.length === 0) return -1
  let lo = 0
  let hi = times.length - 1
  while (lo < hi) {
    const mid = (lo + hi) >> 1
    if ((times[mid] ?? 0) < t) lo = mid + 1
    else hi = mid
  }
  const prev = times[lo - 1]
  if (lo > 0 && prev !== undefined && t - prev < (times[lo] ?? 0) - t) return lo - 1
  return lo
}

export type Level = '' | 'warn' | 'danger'

/** Severity of a reading against its thresholds, for --warn/--danger styling. */
export function level(value: number, warn: number, danger: number): Level {
  if (value >= danger) return 'danger'
  if (value >= warn) return 'warn'
  return ''
}
