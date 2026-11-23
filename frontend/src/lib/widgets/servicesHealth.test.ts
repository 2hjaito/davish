import { describe, expect, it } from 'vitest'

import type { Unit } from '../api/services'
import { groupOf, hasProblem, LIST_MAX, problemCount, summarize } from './servicesHealth'

const u = (name: string, active: string, sub: string): Unit => ({
  name,
  load: 'loaded',
  active,
  sub,
  description: '',
})

describe('groupOf', () => {
  it.each([
    ['active', 'running', 'running'],
    ['active', 'exited', 'exited'],
    ['active', 'waiting', 'exited'],
    ['activating', 'start', 'starting'],
    ['reloading', 'reload', 'starting'],
    ['failed', 'failed', 'bad'],
    ['activating', 'auto-restart', 'bad'],
    ['inactive', 'dead', 'inactive'],
    ['deactivating', 'stop-sigterm', 'inactive'],
    ['', '', 'inactive'],
  ])('%s / %s is %s', (active, sub, want) => {
    expect(groupOf({ active, sub })).toBe(want)
  })

  it('calls a unit that crashed and is waiting to retry bad, not starting', () => {
    expect(groupOf({ active: 'activating', sub: 'auto-restart' })).toBe('bad')
  })
})

describe('summarize', () => {
  it('counts every unit in exactly one group', () => {
    const s = summarize([
      u('nginx.service', 'active', 'running'),
      u('cron.service', 'active', 'running'),
      u('setup.service', 'active', 'exited'),
      u('app.service', 'failed', 'failed'),
      u('worker.service', 'activating', 'auto-restart'),
      u('boot.service', 'activating', 'start'),
      u('old.service', 'inactive', 'dead'),
    ])
    expect(s.total).toBe(7)
    expect(s.counts).toEqual({ running: 2, exited: 1, starting: 1, bad: 2, inactive: 1 })
    expect(Object.values(s.counts).reduce((a, b) => a + b, 0)).toBe(s.total)
    expect(hasProblem(s)).toBe(true)
    expect(problemCount(s)).toBe(2)
  })

  it('names the bad ones without .service, and says why', () => {
    const s = summarize([
      u('app.service', 'failed', 'failed'),
      u('worker.service', 'activating', 'auto-restart'),
    ])
    expect(s.problems).toEqual([
      { name: 'app', why: 'failed' },
      { name: 'worker', why: 'restarting' },
    ])
  })

  it('cuts the names at LIST_MAX and counts the rest', () => {
    const many = Array.from({ length: LIST_MAX + 4 }, (_, i) =>
      u(`s${i}.service`, 'failed', 'failed'),
    )
    const s = summarize(many)
    expect(s.problems).toHaveLength(LIST_MAX)
    expect(s.more).toBe(4)
    expect(problemCount(s)).toBe(LIST_MAX + 4)
  })

  it('has no problem when all is well, and none for no units', () => {
    expect(hasProblem(summarize([u('a.service', 'active', 'running')]))).toBe(false)
    expect(summarize([])).toEqual({
      total: 0,
      counts: { running: 0, exited: 0, starting: 0, bad: 0, inactive: 0 },
      problems: [],
      more: 0,
    })
  })
})
