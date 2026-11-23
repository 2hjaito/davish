import { describe, expect, it } from 'vitest'

import type { Mount } from '../api/events'
import { levelOf, summarizeMounts, WARN_BELOW } from './diskUsage'

const m = (path: string, pct: number, size = 100): Mount => ({
  device: `/dev${path}`,
  path,
  size,
  used: (size * pct) / 100,
  avail: size - (size * pct) / 100,
  pct,
})

describe('levelOf', () => {
  it('is danger from the threshold and warn within ten points under it', () => {
    expect(levelOf(85, 85)).toBe('danger')
    expect(levelOf(99, 85)).toBe('danger')
    expect(levelOf(84, 85)).toBe('warn')
    expect(levelOf(85 - WARN_BELOW, 85)).toBe('warn')
    expect(levelOf(85 - WARN_BELOW - 1, 85)).toBe('')
    expect(levelOf(0, 85)).toBe('')
  })

  it('follows the threshold the person set', () => {
    expect(levelOf(70, 70)).toBe('danger')
    expect(levelOf(70, 95)).toBe('')
  })

  it('has no level for a reading that is not a number', () => {
    expect(levelOf(Number.NaN, 85)).toBe('')
  })
})

describe('summarizeMounts', () => {
  it('puts the fullest mount first, and says which it is', () => {
    const s = summarizeMounts([m('/', 40), m('/var', 91), m('/home', 62)], 85)
    expect(s.rows.map((r) => r.path)).toEqual(['/var', '/home', '/'])
    expect(s.fullest?.path).toBe('/var')
  })

  it('takes the worst level of any mount, not the root only', () => {
    expect(summarizeMounts([m('/', 20), m('/data', 90)], 85).level).toBe('danger')
    expect(summarizeMounts([m('/', 20), m('/data', 80)], 85).level).toBe('warn')
    expect(summarizeMounts([m('/', 20), m('/data', 30)], 85).level).toBe('')
  })

  it('leaves out a mount that has no size, and is empty for none', () => {
    expect(summarizeMounts([m('/proc', 0, 0)], 85).rows).toEqual([])
    expect(summarizeMounts([], 85)).toEqual({ fullest: null, level: '', rows: [] })
  })

  it('orders mounts that are as full by path, so the order does not jump', () => {
    const s = summarizeMounts([m('/b', 50), m('/a', 50)], 85)
    expect(s.rows.map((r) => r.path)).toEqual(['/a', '/b'])
  })
})
