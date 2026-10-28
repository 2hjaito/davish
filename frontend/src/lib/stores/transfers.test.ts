import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { TransferDone, TransferProgress } from '../api/events'
import type { Job } from '../api/transfer'

const handlers: {
  progress?: (p: TransferProgress) => void
  done?: (d: TransferDone) => void
} = {}

vi.mock('../api/runtime', () => ({ hasRuntime: true }))
vi.mock('../api/events', () => ({
  onTransferProgress: (h: (p: TransferProgress) => void) => {
    handlers.progress = h
    return () => {}
  },
  onTransferDone: (h: (d: TransferDone) => void) => {
    handlers.done = h
    return () => {}
  },
}))

const { describeTransfer, landingKey, summarize, transfers } = await import('./transfers.svelte')

const noActions = { showProgress: () => {}, reveal: () => {} }

const job = (id: string, kind: Job['kind'] = 'upload'): Job => ({
  id,
  kind,
  serverID: 's1',
  name: id + '.txt',
  src: '/local/' + id,
  dst: '/srv/www/' + id + '.txt',
  dir: false,
})

describe('transfers', () => {
  const finished: string[] = []
  const started: string[] = []

  beforeEach(() => {
    transfers.list = []
    transfers.landed = {}
    finished.length = 0
    started.length = 0
    transfers.init(
      (j) => started.push(j.id),
      (j) => finished.push(`${j.id}:${j.state}`),
    )
  })

  it('applies progress and completion', () => {
    transfers.add([job('a')])
    handlers.progress?.({ jobID: 'a', done: 50, total: 100, bps: 10 })
    expect(transfers.list[0]).toMatchObject({ done: 50, total: 100, state: 'running' })
    handlers.done?.({ jobID: 'a' })
    expect(transfers.list[0]?.state).toBe('done')
    expect(transfers.landed[landingKey('s1', '/srv/www')]).toBe(1)
    expect(finished).toEqual(['a:done'])
    expect(started).toEqual(['a'])
  })

  it('keeps events that arrive before the job is known', () => {
    handlers.progress?.({ jobID: 'b', done: 10, total: 10, bps: 0 })
    handlers.done?.({ jobID: 'b' })
    transfers.add([job('b')])
    expect(transfers.list[0]).toMatchObject({ done: 10, state: 'done' })
    // Started is reported before finished, so "done" replaces "downloading".
    expect(started).toEqual(['b'])
    expect(finished).toEqual(['b:done'])
  })

  it('records failures with a user message and cancels without one', () => {
    transfers.add([job('c'), job('d')])
    handlers.done?.({ jobID: 'c', error: { code: 'FS_PERMISSION_DENIED', message: 'x' } })
    handlers.done?.({ jobID: 'd', canceled: true })
    expect(transfers.list.map((j) => j.state)).toEqual(['failed', 'canceled'])
    expect(transfers.list[0]?.error).toBe('Không có quyền truy cập.')
    expect(transfers.landed).toEqual({})
  })
})

describe('summarize', () => {
  it('adds up the running jobs of one server', () => {
    const base = { ...job('x'), bps: 0 }
    const list = [
      { ...base, id: '1', done: 30, total: 100, state: 'running' as const },
      { ...base, id: '2', done: 20, total: 100, state: 'running' as const },
      { ...base, id: '3', done: 100, total: 100, state: 'done' as const },
      { ...base, id: '4', serverID: 's2', done: 0, total: 100, state: 'running' as const },
    ]
    expect(summarize(list, 's1')).toEqual({ count: 2, fraction: 0.25 })
    expect(summarize([], 's1')).toEqual({ count: 0, fraction: 0 })
  })
})

describe('describeTransfer', () => {
  const base = { ...job('x'), done: 0, total: 0, bps: 0 }

  it('follows a download from start to finish under one id', () => {
    const revealed: string[] = []
    const actions = { showProgress: () => {}, reveal: (j: { id: string }) => revealed.push(j.id) }
    const running = describeTransfer({ ...base, kind: 'download', state: 'running' }, actions, 1)
    const done = describeTransfer({ ...base, kind: 'download', state: 'done' }, actions, 2)
    expect(running).toMatchObject({ level: 'info', title: 'Đang tải xuống “x.txt”' })
    expect(done).toMatchObject({ level: 'info', title: 'Đã tải xong “x.txt”' })
    expect(done?.id).toBe(running?.id)
    done?.action?.run()
    expect(revealed).toEqual(['x'])
  })

  it('reports every failure', () => {
    expect(
      describeTransfer(
        { ...base, state: 'failed', error: 'Không có quyền truy cập.' },
        noActions,
        1,
      ),
    ).toMatchObject({ level: 'error', title: 'Tải lên “x.txt” thất bại' })
  })

  it('stays quiet when the result is already visible', () => {
    expect(describeTransfer({ ...base, state: 'running' }, noActions)).toBeNull()
    expect(describeTransfer({ ...base, state: 'done' }, noActions)).toBeNull()
    expect(describeTransfer({ ...base, kind: 'download', state: 'canceled' }, noActions)).toBeNull()
  })
})

describe('copies between servers', () => {
  const copy = {
    id: 'c1',
    kind: 'copy' as const,
    serverID: 'b',
    fromServerID: 'a',
    name: 'site',
    src: '/srv/site',
    dst: '/home/tester/site',
    dir: true,
    done: 0,
    total: 0,
    bps: 0,
  }
  const showCopy = vi.fn()
  const actions = {
    showProgress: vi.fn(),
    reveal: vi.fn(),
    showCopy,
    nameOf: (id: string) => (id === 'b' ? 'web-2' : id),
  }

  it('says where it goes, and opens the folder once there', () => {
    const running = describeTransfer({ ...copy, state: 'running' }, actions, 0)
    expect(running?.title).toBe('Đang copy “site”')
    expect(running?.body).toBe('Sang web-2, qua máy này.')
    const done = describeTransfer({ ...copy, state: 'done' }, actions, 0)
    expect(done?.body).toBe('Vào /home/tester trên web-2.')
    done?.action?.run()
    expect(showCopy).toHaveBeenCalled()
    expect(describeTransfer({ ...copy, state: 'failed', error: 'x' }, actions, 0)?.title).toBe(
      'Copy “site” thất bại',
    )
  })
})
