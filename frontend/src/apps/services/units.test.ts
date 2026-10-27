import { describe, expect, it } from 'vitest'

import type { Unit } from '../../lib/api/services'
import { canDo, counts, health, shortName, visible } from './units'

const unit = (name: string, active: string, description = ''): Unit => ({
  name,
  load: 'loaded',
  active,
  sub: '',
  description,
})

const units = [
  unit('nginx.service', 'active', 'A high performance web server'),
  unit('cron.service', 'active', 'Regular background program processing daemon'),
  unit('app.service', 'failed', 'My app'),
  unit('apt-daily.service', 'inactive', 'Daily apt download activities'),
  unit('slow.service', 'activating'),
]

describe('health', () => {
  it.each([
    ['active', 'ok'],
    ['reloading', 'ok'],
    ['activating', 'busy'],
    ['deactivating', 'busy'],
    ['failed', 'failed'],
    ['inactive', 'off'],
    ['maintenance', 'off'],
  ])('%s is %s', (active, want) => expect(health(active)).toBe(want))
})

describe('filters', () => {
  it('counts each filter', () => {
    expect(counts(units)).toEqual({ all: 5, running: 3, failed: 1, stopped: 1 })
  })

  it('searches name and description in a filter', () => {
    expect(visible(units, 'all', 'WEB').map((u) => u.name)).toEqual(['nginx.service'])
    expect(visible(units, 'running', 'apt')).toEqual([])
    expect(visible(units, 'stopped', ' apt ').map((u) => u.name)).toEqual(['apt-daily.service'])
    expect(visible(units, 'failed', '')).toHaveLength(1)
  })
})

describe('canDo', () => {
  it('starts only what is not running', () => {
    expect(canDo('inactive', 'start')).toBe(true)
    expect(canDo('failed', 'start')).toBe(true)
    expect(canDo('active', 'start')).toBe(false)
  })

  it('stops only what runs, restarts anything', () => {
    expect(canDo('active', 'stop')).toBe(true)
    expect(canDo('inactive', 'stop')).toBe(false)
    expect(canDo('failed', 'restart')).toBe(true)
  })
})

it('drops the .service suffix', () => {
  expect(shortName('nginx.service')).toBe('nginx')
  expect(shortName('getty@tty1.service')).toBe('getty@tty1')
  expect(shortName('odd')).toBe('odd')
})
