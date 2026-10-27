import { describe, expect, it } from 'vitest'

import type { Process } from '../../lib/api/monitor'
import { nextOrder, orderProcs, serverSort } from './procSort'

const proc = (pid: number, name: string, cpu: number, mem: number, command = name): Process => ({
  pid,
  ppid: 1,
  user: 'root',
  name,
  command,
  state: 'S',
  cpu,
  mem,
  memPct: 0,
  threads: 1,
  age: 10,
})

const procs = [
  proc(30, 'nginx', 2, 50, 'nginx: worker process'),
  proc(10, 'node', 40, 600, 'node /srv/app/server.js'),
  proc(20, 'Postgres', 2, 400),
]

const pids = (ps: Process[]) => ps.map((p) => p.pid)

describe('orderProcs', () => {
  it('sorts numbers, breaking ties by PID', () => {
    expect(pids(orderProcs(procs, { column: 'cpu', desc: true }, ''))).toEqual([10, 20, 30])
    expect(pids(orderProcs(procs, { column: 'cpu', desc: false }, ''))).toEqual([20, 30, 10])
  })

  it('sorts names without regard to case', () => {
    expect(pids(orderProcs(procs, { column: 'name', desc: false }, ''))).toEqual([30, 10, 20])
  })

  it('filters by name or command line', () => {
    expect(pids(orderProcs(procs, { column: 'pid', desc: false }, ' SERVER.js '))).toEqual([10])
    expect(pids(orderProcs(procs, { column: 'pid', desc: false }, 'n'))).toEqual([10, 30])
  })

  it('leaves the input alone', () => {
    const copy = [...procs]
    orderProcs(procs, { column: 'mem', desc: true }, '')
    expect(procs).toEqual(copy)
  })
})

describe('nextOrder', () => {
  it('flips the sorted column and starts others at their default', () => {
    expect(nextOrder({ column: 'cpu', desc: true }, 'cpu')).toEqual({ column: 'cpu', desc: false })
    expect(nextOrder({ column: 'cpu', desc: true }, 'name')).toEqual({
      column: 'name',
      desc: false,
    })
    expect(nextOrder({ column: 'name', desc: false }, 'mem')).toEqual({ column: 'mem', desc: true })
  })
})

describe('serverSort', () => {
  it('asks the server for the top by memory only when sorting by memory', () => {
    expect(serverSort('mem')).toBe('mem')
    expect(serverSort('cpu')).toBe('cpu')
    expect(serverSort('name')).toBe('cpu')
  })
})
