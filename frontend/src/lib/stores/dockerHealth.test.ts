import { afterEach, beforeEach, describe, expect, it, type Mock, vi } from 'vitest'

import type { Container } from '../api/docker'
import { DockerHealth, MIN_GAP_MS, REFRESH_MS } from './dockerHealth.svelte'

const S = 'server-1'
const T = 'server-2'

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

const GOOD = [c('web', 'running', 'Up 1 hour (healthy)'), c('db', 'running', 'Up 1 hour')]

let clock = 0
let connected = new Set([S, T])
let active = true
let list: Mock<(id: string) => Promise<Container[]>>
let h: DockerHealth

const err = (code: string) => Object.assign(new Error(code), { code })

beforeEach(() => {
  vi.useFakeTimers()
  clock = 1_000_000
  connected = new Set([S, T])
  active = true
  list = vi.fn(async () => GOOD)
  h = new DockerHealth({
    list: (id) => list(id),
    connected: (id) => connected.has(id),
    active: () => active,
    now: () => clock,
  })
})

afterEach(() => vi.useRealTimers())

async function advance(ms: number) {
  clock += ms
  await vi.advanceTimersByTimeAsync(ms)
}

describe('refresh', () => {
  it('reads a connected server and keeps the summary', async () => {
    await h.refresh(S)
    expect(list).toHaveBeenCalledWith(S)
    expect(h.of(S)).toMatchObject({ problem: null, loading: false, at: 1_000_000 })
    expect(h.of(S).summary).toMatchObject({ total: 2, running: 2 })
  })

  it('does not read a server that is not connected', async () => {
    await h.refresh('elsewhere')
    expect(list).not.toHaveBeenCalled()
    expect(h.of('elsewhere').summary).toBeNull()
  })

  it('does not read twice at once, nor twice within the gap', async () => {
    await Promise.all([h.refresh(S), h.refresh(S)])
    expect(list).toHaveBeenCalledTimes(1)
    clock += MIN_GAP_MS - 1
    await h.refresh(S)
    expect(list).toHaveBeenCalledTimes(1)
    clock += 1
    await h.refresh(S)
    expect(list).toHaveBeenCalledTimes(2)
  })

  it('keeps what was read when a later read fails, and says it failed', async () => {
    await h.refresh(S)
    clock += MIN_GAP_MS
    list.mockRejectedValueOnce(err('CONN_UNREACHABLE'))
    await h.refresh(S)
    expect(h.of(S).problem).toBe('failed')
    expect(h.of(S).summary).toMatchObject({ total: 2 })
  })

  it.each([
    ['CMD_MISSING', 'missing'],
    ['SUDO_PASSWORD_REQUIRED', 'needsRoot'],
    ['SUDO_PASSWORD_WRONG', 'needsRoot'],
    ['CMD_NEEDS_ROOT', 'needsRoot'],
    ['INTERNAL', 'failed'],
  ])('maps %s to %s', async (code, problem) => {
    list.mockRejectedValueOnce(err(code))
    await h.refresh(S)
    expect(h.of(S).problem).toBe(problem)
  })

  it('keeps a server apart from another', async () => {
    await h.refresh(S)
    expect(h.of(T).summary).toBeNull()
  })
})

describe('watch', () => {
  it('reads on a timer while Davish is in front, and stops with the last watcher', async () => {
    const stop = h.watch(S)
    await advance(REFRESH_MS)
    expect(list).toHaveBeenCalledTimes(1)
    await advance(REFRESH_MS)
    expect(list).toHaveBeenCalledTimes(2)
    stop()
    await advance(REFRESH_MS * 3)
    expect(list).toHaveBeenCalledTimes(2)
  })

  it('shares one timer between watchers of a server, and reads each server once', async () => {
    const a = h.watch(S)
    const b = h.watch(S)
    const c2 = h.watch(T)
    await advance(REFRESH_MS)
    expect(list.mock.calls.map((x) => x[0]).sort()).toEqual([S, T])
    a()
    await advance(REFRESH_MS)
    expect(list).toHaveBeenCalledTimes(4) // S still has b
    b()
    c2()
    await advance(REFRESH_MS * 2)
    expect(list).toHaveBeenCalledTimes(4)
  })

  it('reads nothing on the timer while Davish is hidden or not in front', async () => {
    h.watch(S)
    active = false
    await advance(REFRESH_MS * 3)
    expect(list).not.toHaveBeenCalled()
    active = true
    await advance(REFRESH_MS)
    expect(list).toHaveBeenCalledTimes(1)
  })

  it('stops asking a server that has no docker, or wants root, until Davish is back in front', async () => {
    h.watch(S)
    list.mockRejectedValueOnce(err('CMD_MISSING'))
    await advance(REFRESH_MS)
    expect(list).toHaveBeenCalledTimes(1)
    await advance(REFRESH_MS * 3)
    expect(list).toHaveBeenCalledTimes(1) // not asked again
    h.onFocus()
    await advance(0)
    expect(list).toHaveBeenCalledTimes(2) // asked once more on coming back
  })

  it('keeps asking after an ordinary failure', async () => {
    h.watch(S)
    list.mockRejectedValueOnce(err('CONN_UNREACHABLE'))
    await advance(REFRESH_MS)
    await advance(REFRESH_MS)
    expect(list).toHaveBeenCalledTimes(2)
  })

  it('does not read a server that is not connected on the timer', async () => {
    connected.delete(S)
    h.watch(S)
    await advance(REFRESH_MS * 2)
    expect(list).not.toHaveBeenCalled()
  })
})

describe('onFocus', () => {
  it('reads every watched server, within the gap rule', async () => {
    h.watch(S)
    h.watch(T)
    h.onFocus()
    await advance(0)
    expect(list).toHaveBeenCalledTimes(2)
    h.onFocus()
    await advance(0)
    expect(list).toHaveBeenCalledTimes(2) // just read
  })

  it('reads nothing for a server nobody watches', async () => {
    h.onFocus()
    await advance(0)
    expect(list).not.toHaveBeenCalled()
  })
})
