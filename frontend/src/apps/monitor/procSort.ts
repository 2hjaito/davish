import type { Process, ProcessSort } from '../../lib/api/monitor'

export type ProcColumn = 'pid' | 'name' | 'user' | 'cpu' | 'mem' | 'threads' | 'age'

export interface ProcOrder {
  column: ProcColumn
  desc: boolean
}

/** Numbers start with the largest, like Activity Monitor; text from A. */
export function defaultDesc(column: ProcColumn): boolean {
  return column === 'cpu' || column === 'mem' || column === 'threads' || column === 'age'
}

/** Clicking the sorted column flips it; another column starts at its default. */
export function nextOrder(cur: ProcOrder, column: ProcColumn): ProcOrder {
  return cur.column === column ? { column, desc: !cur.desc } : { column, desc: defaultDesc(column) }
}

/**
 * The server only returns the top 50, so it must pick them by what the user
 * cares about: memory when sorting by memory, CPU otherwise.
 */
export function serverSort(column: ProcColumn): ProcessSort {
  return column === 'mem' ? 'mem' : 'cpu'
}

const collator = new Intl.Collator('vi', { numeric: true, sensitivity: 'base' })

function compare(a: Process, b: Process, column: ProcColumn): number {
  switch (column) {
    case 'name':
      return collator.compare(a.name, b.name)
    case 'user':
      return collator.compare(a.user, b.user)
    default:
      return a[column] - b[column]
  }
}

/** Filters by name or command line, then sorts; ties keep PID order. */
export function orderProcs(procs: Process[], order: ProcOrder, query: string): Process[] {
  const q = query.trim().toLowerCase()
  const shown = q
    ? procs.filter((p) => p.name.toLowerCase().includes(q) || p.command.toLowerCase().includes(q))
    : [...procs]
  const sign = order.desc ? -1 : 1
  return shown.sort((a, b) => sign * compare(a, b, order.column) || a.pid - b.pid)
}
