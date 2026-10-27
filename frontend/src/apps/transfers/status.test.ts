import { describe, expect, it } from 'vitest'

import type { Transfer } from '../../lib/stores/transfers.svelte'
import { percent, statusText, timeLeft } from './status'

const job = (over: Partial<Transfer>): Transfer => ({
  id: 'j',
  kind: 'upload',
  serverID: 's',
  name: 'a.iso',
  src: '/a.iso',
  dst: '/srv/a.iso',
  dir: false,
  done: 0,
  total: 0,
  bps: 0,
  state: 'running',
  ...over,
})

describe('timeLeft', () => {
  it('counts seconds, then minutes and hours', () => {
    expect(timeLeft({ done: 0, total: 100, bps: 10 })).toBe('10 giây')
    expect(timeLeft({ done: 0, total: 6000, bps: 10 })).toBe('10 phút')
    expect(timeLeft({ done: 0, total: 3_900_000, bps: 1000 })).toBe('1 giờ 5 phút')
  })

  it('is empty without a speed or a size', () => {
    expect(timeLeft({ done: 0, total: 100, bps: 0 })).toBe('')
    expect(timeLeft({ done: 10, total: 0, bps: 5 })).toBe('')
  })
})

describe('statusText', () => {
  it('describes each state', () => {
    expect(statusText(job({}))).toBe('Đang chờ…')
    expect(statusText(job({ done: 1024, total: 4096, bps: 1024 }))).toBe(
      '1 KB / 4 KB · 1 KB/s · còn 3 giây',
    )
    expect(statusText(job({ done: 2048 }))).toBe('2 KB')
    expect(statusText(job({ state: 'done', total: 4096 }))).toBe('Đã xong · 4 KB')
    expect(statusText(job({ state: 'canceled' }))).toBe('Đã hủy')
    expect(statusText(job({ state: 'failed', error: 'Không có quyền truy cập.' }))).toBe(
      'Không có quyền truy cập.',
    )
  })
})

describe('percent', () => {
  it('follows bytes and fills on success', () => {
    expect(percent(job({ done: 25, total: 100 }))).toBe(25)
    expect(percent(job({ done: 5 }))).toBe(0)
    expect(percent(job({ state: 'done' }))).toBe(100)
  })
})
