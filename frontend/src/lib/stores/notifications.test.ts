import { describe as suite, expect, it, vi } from 'vitest'

import type { NotifyNew } from '../api/events'

const api = vi.hoisted(() => ({
  off: vi.fn(),
  emit: undefined as ((p: NotifyNew) => void) | undefined,
}))

vi.mock('../api/runtime', () => ({ hasRuntime: true }))
vi.mock('../api/events', () => ({
  onNotifyNew: (h: (p: NotifyNew) => void) => {
    api.emit = h
    return api.off
  },
}))

const { describe, notifications } = await import('./notifications.svelte')

const mem: NotifyNew = {
  id: 's1:mem',
  level: 'warn',
  serverID: 's1',
  kind: 'mem-high',
  value: 93.4,
  threshold: 90,
}

suite('describe', () => {
  it('writes memory alerts', () => {
    const n = describe(mem, 'web-1')
    expect(n.title).toBe('RAM trên web-1 sắp đầy')
    expect(n.body).toBe('Đang dùng 93%, trên ngưỡng 90%.')
  })

  it('names the mount of disk alerts', () => {
    const n = describe(
      { ...mem, id: 's1:disk:/data', kind: 'disk-high', mount: '/data', value: 88, threshold: 85 },
      'web-1',
    )
    expect(n.title).toBe('Ổ /data trên web-1 sắp đầy')
    expect(n.body).toBe('Đang dùng 88%, trên ngưỡng 85%.')
  })
})

suite('notifications', () => {
  it('adds notify:new on top and replaces a repeat of the same id', () => {
    const stop = notifications.init((id) => (id === 's1' ? 'web-1' : id))
    api.emit?.(mem)
    api.emit?.({ ...mem, id: 's1:disk:/', kind: 'disk-high', mount: '/', value: 86 })
    api.emit?.({ ...mem, value: 97 })

    expect(notifications.list.map((n) => [n.id, n.body])).toEqual([
      ['s1:mem', 'Đang dùng 97%, trên ngưỡng 90%.'],
      ['s1:disk:/', 'Đang dùng 86%, trên ngưỡng 90%.'],
    ])
    expect(notifications.unread).toBe(2)

    notifications.dismiss('s1:mem')
    expect(notifications.list.map((n) => n.id)).toEqual(['s1:disk:/'])

    stop()
    expect(api.off).toHaveBeenCalledOnce()
  })
})
