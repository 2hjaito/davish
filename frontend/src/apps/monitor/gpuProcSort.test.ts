import { describe, expect, it } from 'vitest'

import type { GPUProcess } from '../../lib/api/monitor'
import { DEFAULT_GPU_ORDER, nextGpuOrder, oneLine, orderGpuProcs } from './gpuProcSort'

const MB = 1024 ** 2

const row = (pid: number, gpuMem: number, over: Partial<GPUProcess> = {}): GPUProcess => ({
  gpu: 0,
  pid,
  type: 'C',
  gpuUtil: 0,
  gpuMem,
  gpuMemPct: gpuMem < 0 ? -1 : 1,
  user: 'root',
  name: `p${pid}`,
  command: `p${pid}`,
  cpu: 0,
  mem: 0,
  ...over,
})

const pids = (rows: GPUProcess[]) => rows.map((r) => r.pid)

describe('orderGpuProcs', () => {
  const rows = [row(1, 552 * MB), row(2, -1), row(3, 21382 * MB), row(4, 2932 * MB)]

  it('puts the most GPU memory first by default, unknown last', () => {
    expect(pids(orderGpuProcs(rows, DEFAULT_GPU_ORDER))).toEqual([3, 4, 1, 2])
  })

  it('keeps unknown readings last when sorting up', () => {
    expect(pids(orderGpuProcs(rows, { column: 'gpuMem', desc: false }))).toEqual([1, 4, 3, 2])
  })

  it('tells the same process on two GPUs apart by GPU', () => {
    const both = [row(7, MB, { gpu: 1 }), row(7, MB, { gpu: 0 })]
    expect(orderGpuProcs(both, DEFAULT_GPU_ORDER).map((r) => r.gpu)).toEqual([0, 1])
  })

  it('sorts text columns by name, empty users last', () => {
    const users = [row(1, 0, { user: 'rst' }), row(2, 0, { user: '' }), row(3, 0, { user: 'ml' })]
    expect(pids(orderGpuProcs(users, { column: 'user', desc: false }))).toEqual([3, 1, 2])
  })

  it('does not change the rows it was given', () => {
    const copy = [...rows]
    orderGpuProcs(rows, DEFAULT_GPU_ORDER)
    expect(rows).toEqual(copy)
  })
})

describe('nextGpuOrder', () => {
  it('flips the sorted column', () => {
    expect(nextGpuOrder({ column: 'gpuMem', desc: true }, 'gpuMem')).toEqual({
      column: 'gpuMem',
      desc: false,
    })
  })

  it('starts text from A and numbers from the largest', () => {
    expect(nextGpuOrder(DEFAULT_GPU_ORDER, 'user')).toEqual({ column: 'user', desc: false })
    expect(nextGpuOrder(DEFAULT_GPU_ORDER, 'cpu')).toEqual({ column: 'cpu', desc: true })
  })
})

describe('oneLine', () => {
  it('keeps a multi-line command on one row', () => {
    expect(oneLine('python3 -c\nimport sys, os, logging\nfrom whisper_liv')).toBe(
      'python3 -c import sys, os, logging from whisper_liv',
    )
  })

  it('collapses tabs and runs of spaces', () => {
    expect(oneLine('  a\t\tb   c \r\n')).toBe('a b c')
  })
})
