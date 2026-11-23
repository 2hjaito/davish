import { describe, expect, it } from 'vitest'

import type { Container } from '../api/docker'
import {
  DOT_MAX,
  dotOf,
  hasProblem,
  healthOf,
  LIST_MAX,
  problemCount,
  summarize,
} from './dockerHealth'

const c = (name: string, state: string, status: string): Container => ({
  id: name,
  name,
  image: 'img',
  state,
  status,
  ports: '',
  command: '',
  created: '',
  runningFor: '',
})

describe('healthOf', () => {
  it.each([
    ['Up 3 hours (healthy)', 'healthy'],
    ['Up 2 minutes (unhealthy)', 'unhealthy'],
    ['Up 5 seconds (health: starting)', 'starting'],
    ['Up 3 hours', 'none'],
    ['Exited (1) 2 minutes ago', 'none'],
    ['Restarting (1) 5 seconds ago', 'none'],
    ['', 'none'],
    // a name or a word that merely looks like one is not a healthcheck result
    ['Up 3 hours (paused)', 'none'],
    ['Up 3 hours healthy', 'none'],
  ])('%j is %s', (status, want) => {
    expect(healthOf({ status })).toBe(want)
  })
})

describe('summarize', () => {
  it('counts running containers and the total', () => {
    const s = summarize([c('a', 'running', 'Up 1 hour'), c('b', 'exited', 'Exited (0) 2 days ago')])
    expect(s).toMatchObject({ total: 2, running: 1, unhealthy: [], restarting: [], starting: 0 })
    expect(hasProblem(s)).toBe(false)
  })

  it('names the ones that fail their healthcheck or keep restarting', () => {
    const s = summarize([
      c('web', 'running', 'Up 1 hour (unhealthy)'),
      c('db', 'running', 'Up 1 hour (healthy)'),
      c('worker', 'restarting', 'Restarting (1) 5 seconds ago'),
      c('cache', 'running', 'Up 5 seconds (health: starting)'),
    ])
    expect(s.unhealthy).toEqual(['web'])
    expect(s.restarting).toEqual(['worker'])
    expect(s.starting).toBe(1)
    expect(s.running).toBe(3)
    expect(hasProblem(s)).toBe(true)
    expect(problemCount(s)).toBe(2)
  })

  it('cuts the names at LIST_MAX and counts the rest', () => {
    const many = Array.from({ length: LIST_MAX + 7 }, (_, i) =>
      c(`c${i}`, 'running', 'Up 1 hour (unhealthy)'),
    )
    const s = summarize(many)
    expect(s.unhealthy).toHaveLength(LIST_MAX)
    expect(s.more).toBe(7)
    expect(problemCount(s)).toBe(LIST_MAX + 7)
  })

  it('is empty for no containers', () => {
    expect(summarize([])).toEqual({
      dots: [],
      total: 0,
      running: 0,
      unhealthy: [],
      restarting: [],
      starting: 0,
      more: 0,
    })
  })
})

describe('dotOf', () => {
  it.each([
    ['running', 'Up 1 hour', 'ok'],
    ['running', 'Up 1 hour (healthy)', 'ok'],
    ['running', 'Up 5 seconds (health: starting)', 'starting'],
    ['running', 'Up 1 hour (unhealthy)', 'bad'],
    ['restarting', 'Restarting (1) 5 seconds ago', 'bad'],
    ['exited', 'Exited (0) 2 days ago', 'stopped'],
    ['created', 'Created', 'stopped'],
    ['paused', 'Up 1 hour (Paused)', 'stopped'],
    ['dead', '', 'stopped'],
  ])('%s with %j is %s', (state, status, want) => {
    expect(dotOf({ state, status })).toBe(want)
  })
})

describe('the dots of a summary', () => {
  it("keep docker's order, so a dot stays where it was as states change", () => {
    const s = summarize([
      c('a', 'running', 'Up 1 hour'),
      c('b', 'exited', 'Exited (0) 1 hour ago'),
      c('c', 'running', 'Up 1 hour (unhealthy)'),
      c('d', 'running', 'Up 5 seconds (health: starting)'),
    ])
    expect(s.dots).toEqual(['ok', 'stopped', 'bad', 'starting'])
  })

  it('stop at DOT_MAX while the counts go on', () => {
    const many = Array.from({ length: DOT_MAX + 9 }, (_, i) => c(`c${i}`, 'running', 'Up 1 hour'))
    const s = summarize(many)
    expect(s.dots).toHaveLength(DOT_MAX)
    expect(s.total).toBe(DOT_MAX + 9)
    expect(s.running).toBe(DOT_MAX + 9)
  })
})
