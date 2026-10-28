// Pure helpers of the Docker app: filters, states and times.
import type { Container, ContainerAction } from '../../lib/api/docker'

export type Filter = 'all' | 'running' | 'restarting' | 'stopped'

export const FILTERS: readonly Filter[] = ['all', 'running', 'restarting', 'stopped']

/** The dot color of a state. A restart loop is as bad as a failure. */
export type Health = 'ok' | 'busy' | 'failed' | 'off'

export function health(state: string): Health {
  switch (state) {
    case 'running':
      return 'ok'
    case 'paused':
    case 'removing':
      return 'busy'
    case 'restarting':
    case 'dead':
      return 'failed'
  }
  return 'off'
}

export function matches(c: Container, filter: Filter): boolean {
  switch (filter) {
    case 'running':
      return c.state === 'running' || c.state === 'paused'
    case 'restarting':
      return c.state === 'restarting'
    case 'stopped':
      return c.state === 'exited' || c.state === 'created' || c.state === 'dead'
  }
  return true
}

export function counts(list: readonly Container[]): Record<Filter, number> {
  const c: Record<Filter, number> = { all: 0, running: 0, restarting: 0, stopped: 0 }
  for (const x of list) for (const f of FILTERS) if (matches(x, f)) c[f]++
  return c
}

/** Containers in filter whose name or image contains query, any case. */
export function visible(list: readonly Container[], filter: Filter, query: string): Container[] {
  const q = query.trim().toLowerCase()
  return list.filter(
    (c) =>
      matches(c, filter) &&
      (!q || c.name.toLowerCase().includes(q) || c.image.toLowerCase().includes(q)),
  )
}

/** Which actions make sense in a state; the others are disabled. */
export function canDo(state: string, action: ContainerAction): boolean {
  const running = state === 'running' || state === 'restarting' || state === 'paused'
  switch (action) {
    case 'start':
      return !running && state !== 'removing'
    case 'stop':
      return running
  }
  return state !== 'removing'
}

/** Seconds from an RFC 3339 time to now; 0 when unknown or in the future. */
export function since(rfc3339: string, now: number): number {
  const t = Date.parse(rfc3339)
  return Number.isNaN(t) ? 0 : Math.max(0, Math.floor((now - t) / 1000))
}

/** "15:54:02" in local time, for a log line; the raw text when unparsable. */
export function clock(rfc3339: string): string {
  const t = Date.parse(rfc3339)
  if (Number.isNaN(t)) return rfc3339
  const d = new Date(t)
  const p = (n: number) => String(n).padStart(2, '0')
  return `${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`
}
