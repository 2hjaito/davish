import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { SysmonTick } from '../api/events'

const api = vi.hoisted(() => ({
  watch: vi.fn<(id: string) => Promise<void>>(() => Promise.resolve()),
  unwatch: vi.fn<(id: string) => Promise<void>>(() => Promise.resolve()),
  off: vi.fn(),
  emit: undefined as ((t: SysmonTick) => void) | undefined,
}))

vi.mock('../api/runtime', () => ({ hasRuntime: true }))
vi.mock('../api/monitor', () => ({ monitorWatch: api.watch, monitorUnwatch: api.unwatch }))
vi.mock('../api/events', () => ({
  onSysmonTick: (h: (t: SysmonTick) => void) => {
    api.emit = h
    return api.off
  },
}))

const { HISTORY_MS, metrics } = await import('./metrics.svelte')
const { firstMockTick } = await import('./mockMetrics')

const tick = (serverID: string, cpu: number): SysmonTick => ({
  ...firstMockTick(),
  serverID,
  cpu,
})

describe('metrics', () => {
  beforeEach(() => vi.clearAllMocks())

  it('watches a server once for several watchers and stops with the last', () => {
    const a = metrics.watch('s1')
    const b = metrics.watch('s1')
    expect(api.watch).toHaveBeenCalledTimes(1)

    api.emit?.(tick('s1', 12.5))
    expect(metrics.latest('s1')?.cpu).toBe(12.5)

    a()
    a() // a second call of the same stop function is a no-op
    expect(api.unwatch).not.toHaveBeenCalled()
    expect(metrics.latest('s1')).toBeDefined()

    b()
    expect(api.unwatch).toHaveBeenCalledWith('s1')
    expect(metrics.latest('s1')).toBeUndefined()
    expect(api.off).toHaveBeenCalledTimes(1)
  })

  it('ignores ticks for servers nobody watches', () => {
    const stop = metrics.watch('s1')
    api.emit?.(tick('s2', 50))
    expect(metrics.latest('s2')).toBeUndefined()
    stop()
    api.emit?.(tick('s1', 50))
    expect(metrics.latest('s1')).toBeUndefined()
  })

  it('keeps listening while another server is still watched', () => {
    const s1 = metrics.watch('s1')
    const s2 = metrics.watch('s2')
    s1()
    expect(api.off).not.toHaveBeenCalled()
    api.emit?.(tick('s2', 7))
    expect(metrics.latest('s2')?.cpu).toBe(7)
    s2()
    expect(api.off).toHaveBeenCalledTimes(1)
  })

  it('keeps history for the longest chart range only', () => {
    const stop = metrics.watch('s1')
    const t0 = 1_000_000
    metrics.push('s1', tick('s1', 1), t0)
    metrics.push('s1', tick('s1', 2), t0 + 2000)
    metrics.push('s1', tick('s1', 3), t0 + HISTORY_MS + 1000)
    expect(metrics.history('s1').map((s) => s.cpu)).toEqual([2, 3])
    expect(metrics.latest('s1')?.at).toBe(t0 + HISTORY_MS + 1000)

    metrics.push('s1', tick('s1', 4), t0 + 3 * HISTORY_MS)
    expect(metrics.history('s1').map((s) => s.cpu)).toEqual([4])
    stop()
    expect(metrics.history('s1')).toEqual([])
  })
})
