import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('../wm/intents.svelte', () => ({ focusTerminal: vi.fn() }))

const { deploys, durations, HISTORY_MAX, parseMark } = await import('./deploys.svelte')
const { notifications } = await import('./notifications.svelte')

const N = 'ab12'

describe('parseMark', () => {
  it('reads begin, step and end', () => {
    expect(parseMark(`davish-deploy;${N};begin;3`)).toEqual({ nonce: N, kind: 'begin', steps: 3 })
    expect(parseMark(`davish-deploy;${N};step;2`)).toEqual({ nonce: N, kind: 'step', step: 2 })
    expect(parseMark(`davish-deploy;${N};end;0;71d4c4c`)).toEqual({
      nonce: N,
      kind: 'end',
      exit: 0,
      commit: '71d4c4c',
    })
    expect(parseMark(`davish-deploy;${N};end;3`)).toEqual({ nonce: N, kind: 'end', exit: 3 })
  })

  it('drops anything else', () => {
    for (const bad of [
      'other;ab;begin;1',
      'davish-deploy;XYZ;begin;1',
      'davish-deploy;ab;step;0',
      'davish-deploy;ab;step;x',
      'davish-deploy;ab;jump;1',
      'davish-deploy;ab',
    ])
      expect(parseMark(bad), bad).toBeNull()
    expect(parseMark('davish-deploy;ab;end;0;<b>x</b>')).toEqual({
      nonce: 'ab',
      kind: 'end',
      exit: 0,
    })
  })
})

describe('deploys', () => {
  beforeEach(() => {
    deploys.runs = {}
    deploys.history = {}
    notifications.list = []
  })

  const start = (steps = ['git pull', 'npm ci', 'restart']) =>
    deploys.watch('s1', '/srv/app', N, steps)
  const mark = (data: string, at: number) => {
    const m = parseMark(`davish-deploy;${N};${data}`)
    if (m) deploys.apply(m, at)
  }

  it('follows a run to the end and keeps a record', () => {
    start()
    expect(deploys.runs[N]?.state).toBe('waiting')
    mark('begin;3', 1000)
    mark('step;1', 1000)
    mark('step;2', 3000)
    mark('step;3', 7000)
    expect(deploys.runs[N]?.current).toBe(3)
    mark('end;0;abc1234', 8000)
    const run = deploys.runs[N]
    expect(run?.state).toBe('ok')
    expect(run && durations(run, 8000)).toEqual([2000, 4000, 1000])
    expect(deploys.historyOf('s1', '/srv/app')).toEqual([
      {
        at: 1000,
        state: 'ok',
        step: undefined,
        exit: 0,
        commit: 'abc1234',
        durations: [2000, 4000, 1000],
      },
    ])
    expect(notifications.list[0]?.level).toBe('info')
  })

  it('marks the failing step', () => {
    start()
    mark('begin;3', 0)
    mark('step;1', 0)
    mark('step;2', 10)
    mark('end;3', 20)
    expect(deploys.runs[N]).toMatchObject({ state: 'failed', current: 2, exit: 3 })
    expect(deploys.historyOf('s1', '/srv/app')[0]).toMatchObject({
      state: 'failed',
      step: 2,
      exit: 3,
    })
    expect(notifications.list[0]?.level).toBe('error')
  })

  it('ignores marks of another nonce, out of order, or after the end', () => {
    start()
    deploys.apply({ nonce: 'ffff', kind: 'step', step: 1 })
    expect(deploys.runs[N]?.state).toBe('waiting')
    mark('step;1', 0)
    mark('step;3', 0)
    mark('step;1', 0)
    expect(deploys.runs[N]?.current).toBe(1)
    mark('end;1', 0)
    mark('end;0', 0)
    expect(deploys.runs[N]?.state).toBe('failed')
    expect(deploys.historyOf('s1', '/srv/app')).toHaveLength(1)
  })

  it('an end of 0 before the last step is not a success', () => {
    start()
    mark('step;1', 0)
    mark('end;0', 0)
    expect(deploys.runs[N]?.state).toBe('failed')
  })

  it('stops a running run and forgets one never started', () => {
    start()
    deploys.stop(N)
    expect(deploys.runs[N]).toBeUndefined()
    start()
    mark('step;1', 0)
    deploys.stop(N, 50)
    expect(deploys.runs[N]?.state).toBe('stopped')
    expect(deploys.historyOf('s1', '/srv/app')[0]).toMatchObject({ state: 'stopped', step: 1 })
    deploys.forget(N)
    expect(deploys.runs[N]).toBeUndefined()
  })

  it(`keeps the last ${HISTORY_MAX} runs`, () => {
    for (let i = 0; i < HISTORY_MAX + 3; i++) {
      start(['x'])
      mark('step;1', i)
      mark('end;0', i)
    }
    expect(deploys.historyOf('s1', '/srv/app')).toHaveLength(HISTORY_MAX)
  })
})
