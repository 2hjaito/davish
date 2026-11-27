import type { GPUProcess } from '../../lib/api/monitor'
import { locale } from '../../lib/i18n'

export type GpuProcColumn =
  'pid' | 'user' | 'gpu' | 'type' | 'gpuUtil' | 'gpuMem' | 'cpu' | 'mem' | 'command'

export interface GpuProcOrder {
  column: GpuProcColumn
  desc: boolean
}

/** Like nvtop: whoever holds the most GPU memory first. */
export const DEFAULT_GPU_ORDER: GpuProcOrder = { column: 'gpuMem', desc: true }

const TEXT: ReadonlySet<GpuProcColumn> = new Set(['user', 'type', 'command'])

/** Clicking the sorted column flips it; another starts from A for text, the largest for numbers. */
export function nextGpuOrder(cur: GpuProcOrder, column: GpuProcColumn): GpuProcOrder {
  return cur.column === column ? { column, desc: !cur.desc } : { column, desc: !TEXT.has(column) }
}

const collator = new Intl.Collator(locale, { numeric: true, sensitivity: 'base' })

/** -1 and empty text are readings nobody gave. */
function unknown(r: GPUProcess, column: GpuProcColumn): boolean {
  const v = r[column]
  return typeof v === 'number' ? v < 0 : v === ''
}

/**
 * Sorts a copy. Unknown readings stay at the bottom whichever way the column
 * runs; ties keep GPU, then PID order, so a process on two GPUs stays put.
 */
export function orderGpuProcs(rows: GPUProcess[], order: GpuProcOrder): GPUProcess[] {
  const { column } = order
  const sign = order.desc ? -1 : 1
  return [...rows].sort((a, b) => {
    const ua = unknown(a, column)
    const ub = unknown(b, column)
    let c = Number(ua) - Number(ub)
    if (c === 0 && !ua) {
      const va = a[column]
      const vb = b[column]
      c = sign * (typeof va === 'number' ? va - (vb as number) : collator.compare(va, vb as string))
    }
    return c || a.gpu - b.gpu || a.pid - b.pid
  })
}

/** A command line on one row: a `python3 -c` script's newlines become spaces. */
export function oneLine(s: string): string {
  return s.replace(/\s+/g, ' ').trim()
}
