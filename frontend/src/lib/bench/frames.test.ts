import { describe, expect, it } from 'vitest'

import { FRAME_BUDGET_MS, MISSED_FROM, resultLine, summarize } from './frames'

describe('summarize', () => {
  it('is all zero for no frames', () => {
    expect(summarize([])).toEqual({ frames: 0, meanMs: 0, p95Ms: 0, maxMs: 0, overMs: 0 })
  })

  it('counts nothing over budget for a steady 60 fps', () => {
    const s = summarize(Array.from({ length: 60 }, () => FRAME_BUDGET_MS))
    expect(s.frames).toBe(60)
    expect(s.meanMs).toBeCloseTo(FRAME_BUDGET_MS)
    expect(s.overMs).toBeCloseTo(0)
  })

  it('does not count a frame that is a millisecond long as a miss', () => {
    const s = summarize(Array.from({ length: 180 }, () => 17))
    expect(s.overMs).toBe(0)
    expect(s.maxMs).toBe(17)
  })

  it('sums the time past the budget of the frames that missed it', () => {
    // 16.67 (on budget), 33.33 (a miss: 16.67 over), 50 (a miss: 33.33 over)
    const s = summarize([FRAME_BUDGET_MS, FRAME_BUDGET_MS * 2, FRAME_BUDGET_MS * 3])
    expect(s.overMs).toBeCloseTo(FRAME_BUDGET_MS * 3)
    expect(s.maxMs).toBeCloseTo(FRAME_BUDGET_MS * 3)
    expect(s.meanMs).toBeCloseTo(FRAME_BUDGET_MS * 2)
  })

  it('misses from MISSED_FROM budgets, not before', () => {
    expect(summarize([FRAME_BUDGET_MS * MISSED_FROM]).overMs).toBe(0)
    expect(summarize([FRAME_BUDGET_MS * MISSED_FROM + 1]).overMs).toBeGreaterThan(0)
  })

  it('takes the budget as a parameter', () => {
    // 31 and 45 miss a budget of 20 (from 30): 11 and 25 over
    expect(summarize([10, 31, 45], 20).overMs).toBe(36)
  })
})

describe('resultLine', () => {
  const s = { frames: 180, meanMs: 16.7, p95Ms: 18.2, maxMs: 40.04, overMs: 12.34 }

  it('is key=value pairs on one line', () => {
    expect(resultLine('scroll', 'light', s, 2)).toBe(
      'scenario=scroll preset=light frames=180 mean_ms=16.7 p95_ms=18.2 max_ms=40.0 over_ms=12.3 long_tasks=2',
    )
  })

  it('says na where long tasks cannot be seen', () => {
    expect(resultLine('idle', 'balanced', s, null)).toContain('long_tasks=na')
  })
})
