import { describe, expect, it } from 'vitest'

import type { Sample } from '../stores/metrics.svelte'
import { firstMockTick } from '../stores/mockMetrics'
import { DEFAULT_STATS, STAT_IDS, statView, statViews } from './menuStats'

const sample = (over: Partial<Sample> = {}): Sample => ({ ...firstMockTick(), at: 0, ...over })

describe('statViews', () => {
  it('keeps the bar order whatever order they were picked in', () => {
    const ids = statViews(['net', 'cpu', 'disk'], sample()).map((v) => v.id)
    expect(ids).toEqual(['cpu', 'disk', 'net'])
  })

  it('skips GPU on a server without one, and swap without swap', () => {
    const ids = statViews(STAT_IDS, sample({ gpus: [], swapTotal: 0 })).map((v) => v.id)
    expect(ids).not.toContain('gpu')
    expect(ids).not.toContain('swap')
    expect(ids).toContain('uptime')
  })

  it('shows GPU by default once the server has one', () => {
    const gpu = { index: 0, name: 'L4', util: 40, memUsed: 1, memTotal: 2, temp: 50, power: 30 }
    const ids = statViews(DEFAULT_STATS, sample({ gpus: [gpu] })).map((v) => v.id)
    expect(ids).toContain('gpu')
  })
})

describe('statView', () => {
  it('fills the ring by share and keeps it within a full turn', () => {
    expect(statView('cpu', sample({ cpu: 25 }))?.ring).toBe(0.25)
    expect(statView('cpu', sample({ cpu: 140 }))?.ring).toBe(1)
    expect(statView('cpu', sample({ cpu: Number.NaN }))?.ring).toBe(0)
  })

  it('draws rates and uptime with an icon instead of a ring', () => {
    for (const id of ['net', 'io', 'uptime'] as const) {
      expect(statView(id, sample())?.ring).toBeNull()
    }
  })

  it('warns when load passes the core count', () => {
    const cores = [10, 10, 10, 10]
    expect(statView('load', sample({ cores, load1: 3 }))?.level).toBe('')
    expect(statView('load', sample({ cores, load1: 4.5 }))?.level).toBe('warn')
    expect(statView('load', sample({ cores, load1: 7 }))?.level).toBe('danger')
  })

  it('gives every reading with history a chart, and uptime none', () => {
    for (const id of STAT_IDS) {
      const v = statView(id, sample({ gpus: [] }))
      if (!v) continue
      expect(Boolean(v.chart)).toBe(id !== 'uptime')
    }
  })
})
