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

const { BANNER_MS, describe, MAX_BANNERS, notifications } = await import('./notifications.svelte')

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

  it('links health alerts to the logs of what failed', () => {
    const showLogs = vi.fn()
    const failed = describe(
      {
        ...mem,
        id: 's1:svc:nginx.service',
        level: 'error',
        kind: 'service-failed',
        target: 'nginx.service',
      },
      'web-1',
      0,
      showLogs,
    )
    expect(failed.title).toBe('Dịch vụ nginx trên web-1 bị lỗi')
    failed.action?.run()
    expect(showLogs).toHaveBeenLastCalledWith('s1', { kind: 'journal', target: 'nginx.service' })

    const loop = describe(
      { ...mem, id: 's1:ctr:api', kind: 'container-restarting', target: 'api', value: 12 },
      'web-1',
      0,
      showLogs,
    )
    expect(loop.title).toBe('api trên web-1 đang khởi động lại liên tục')
    expect(loop.body).toBe('Container đã khởi động lại 12 lần.')
    loop.action?.run()
    expect(showLogs).toHaveBeenLastCalledWith('s1', { kind: 'docker', target: 'api' })

    expect(
      describe({ ...mem, kind: 'service-failed', target: 'x.service' }, 'w').action,
    ).toBeFalsy()
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

suite('banners', () => {
  const note = (id: string, level: 'info' | 'warn' | 'error' = 'info') => ({
    id,
    level,
    title: id,
    body: '',
    serverID: 's1',
    at: 0,
  })

  it('shows a new notification for a while, then keeps it only in the list', () => {
    vi.useFakeTimers()
    notifications.list = []
    notifications.bannerIds = []
    notifications.add(note('a'))
    expect(notifications.banners.map((n) => n.id)).toEqual(['a'])
    vi.advanceTimersByTime(BANNER_MS.info)
    expect(notifications.banners).toEqual([])
    expect(notifications.list.map((n) => n.id)).toEqual(['a'])
    vi.useRealTimers()
  })

  it('keeps a banner while hovered and drops it when dismissed', () => {
    vi.useFakeTimers()
    notifications.list = []
    notifications.bannerIds = []
    notifications.add(note('b', 'error'))
    notifications.holdBanner('b')
    vi.advanceTimersByTime(BANNER_MS.error * 2)
    expect(notifications.bannerIds).toEqual(['b'])
    notifications.releaseBanner('b')
    vi.advanceTimersByTime(3000)
    expect(notifications.bannerIds).toEqual([])
    notifications.add(note('c'))
    notifications.dismiss('c')
    expect(notifications.bannerIds).toEqual([])
    vi.useRealTimers()
  })

  it('shows at most a few, newest first, and re-shows a replaced one', () => {
    notifications.list = []
    notifications.bannerIds = []
    for (const id of ['1', '2', '3', '4']) notifications.add(note(id))
    expect(notifications.bannerIds).toEqual(['4', '3', '2'].slice(0, MAX_BANNERS))
    notifications.add(note('2'))
    expect(notifications.bannerIds[0]).toBe('2')
  })
})

suite('disk alert', () => {
  it('opens the Disk app at the full mount', () => {
    const opened: string[] = []
    const n = describe(
      {
        id: 'd',
        level: 'warn',
        kind: 'disk-high',
        serverID: 's',
        value: 91,
        threshold: 85,
        mount: '/var',
      },
      'web-1',
      0,
      undefined,
      (id, mount) => opened.push(`${id}:${mount}`),
    )
    n.action?.run()
    expect(opened).toEqual(['s:/var'])
  })
})
