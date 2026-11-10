import { describe, expect, it } from 'vitest'
import type { Notification } from '../../lib/stores/notifications.svelte'
import { alertsFor, memPct } from './summary'

const n = (serverID: string, level: Notification['level'], at: number, title: string) =>
  ({ id: title, serverID, level, at, title, body: '' }) as Notification

describe('alertsFor', () => {
  const list = [
    n('a', 'warn', 2, 'disk full'),
    n('b', 'error', 3, 'nginx failed'),
    n('a', 'error', 1, 'api failed'),
    n('a', 'info', 5, 'updates'),
  ]
  it('counts one server, keeps the worst level and the newest title', () => {
    expect(alertsFor(list, 'a')).toEqual({ count: 3, level: 'error', latest: 'updates' })
  })
  it('is empty for a quiet server', () => {
    expect(alertsFor(list, 'c')).toEqual({ count: 0, level: null, latest: '' })
  })
})

describe('memPct', () => {
  it('divides and guards against no total', () => {
    expect(memPct(2, 8)).toBe(25)
    expect(memPct(2, 0)).toBe(0)
  })
})
