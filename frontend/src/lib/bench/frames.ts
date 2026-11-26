// Frame timing for the render bench (scripts/bench.sh --render). The numbers
// are about what a person feels while a window moves or a list scrolls: how
// long frames take and how much time went over the 60 fps budget.
import { frameP95 } from '../perf'

/** One frame at 60 fps, in ms. */
export const FRAME_BUDGET_MS = 1000 / 60

/**
 * A frame counts as missed from this many budgets. Timestamps come in whole
 * milliseconds, so a frame of 17 ms on a 16.7 ms budget is not a miss.
 */
export const MISSED_FROM = 1.5

export interface FrameStats {
  frames: number
  meanMs: number
  p95Ms: number
  maxMs: number
  /** Time past the budget, summed over the frames that missed it (see MISSED_FROM). */
  overMs: number
}

/** Stats of the gaps between frames; all zero for none. */
export function summarize(gaps: readonly number[], budget = FRAME_BUDGET_MS): FrameStats {
  if (gaps.length === 0) return { frames: 0, meanMs: 0, p95Ms: 0, maxMs: 0, overMs: 0 }
  const sum = gaps.reduce((s, g) => s + g, 0)
  return {
    frames: gaps.length,
    meanMs: sum / gaps.length,
    p95Ms: frameP95(gaps),
    maxMs: Math.max(...gaps),
    overMs: gaps.reduce((s, g) => (g > budget * MISSED_FROM ? s + g - budget : s), 0),
  }
}

/**
 * Frame gaps over ms milliseconds, calling step with the elapsed time before
 * each frame is drawn (to move a window or scroll a list). The first gap is
 * dropped: it includes whatever came before.
 */
export function measure(ms: number, step?: (elapsed: number) => void): Promise<number[]> {
  return new Promise((resolve) => {
    const gaps: number[] = []
    const start = performance.now()
    let last = start
    const tick = (now: number) => {
      gaps.push(now - last)
      last = now
      const elapsed = now - start
      if (elapsed < ms) {
        step?.(elapsed)
        requestAnimationFrame(tick)
      } else resolve(gaps.slice(1))
    }
    step?.(0)
    requestAnimationFrame(tick)
  })
}

/** The result line of a scenario, as the script reads it: key=value pairs. */
export function resultLine(
  scenario: string,
  preset: string,
  widgets: number,
  s: FrameStats,
  longTasks: number | null,
  picture = false,
): string {
  const n = (v: number) => v.toFixed(1)
  return [
    `scenario=${scenario}`,
    `preset=${preset}`,
    `widgets=${widgets}`,
    `picture=${picture ? 1 : 0}`,
    `frames=${s.frames}`,
    `mean_ms=${n(s.meanMs)}`,
    `p95_ms=${n(s.p95Ms)}`,
    `max_ms=${n(s.maxMs)}`,
    `over_ms=${n(s.overMs)}`,
    `long_tasks=${longTasks ?? 'na'}`,
  ].join(' ')
}

/**
 * Counts long tasks (over 50 ms on the main thread) from now until stop() is
 * called. Null where the engine has no Long Tasks API, as WebKit does not.
 */
export function watchLongTasks(): () => number | null {
  let count = 0
  try {
    if (!PerformanceObserver.supportedEntryTypes?.includes('longtask')) return () => null
    const obs = new PerformanceObserver((list) => (count += list.getEntries().length))
    obs.observe({ type: 'longtask' })
    return () => {
      obs.disconnect()
      return count
    }
  } catch {
    return () => null
  }
}
