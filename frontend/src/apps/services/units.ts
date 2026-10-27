// Pure helpers of the Services app: filters, states and log highlighting.
import type { ServiceAction, Unit } from '../../lib/api/services'

export type Filter = 'all' | 'running' | 'failed' | 'stopped'

export const FILTERS: readonly Filter[] = ['all', 'running', 'failed', 'stopped']

/** How a unit's state reads at a glance: the dot color and its filter. */
export type Health = 'ok' | 'busy' | 'failed' | 'off'

export function health(active: string): Health {
  switch (active) {
    case 'active':
    case 'reloading':
      return 'ok'
    case 'activating':
    case 'deactivating':
      return 'busy'
    case 'failed':
      return 'failed'
  }
  return 'off'
}

export function matches(u: Unit, filter: Filter): boolean {
  const h = health(u.active)
  switch (filter) {
    case 'running':
      return h === 'ok' || h === 'busy'
    case 'failed':
      return h === 'failed'
    case 'stopped':
      return h === 'off'
  }
  return true
}

export function counts(units: readonly Unit[]): Record<Filter, number> {
  const c: Record<Filter, number> = { all: 0, running: 0, failed: 0, stopped: 0 }
  for (const u of units) for (const f of FILTERS) if (matches(u, f)) c[f]++
  return c
}

/** The name without ".service", as people say it. */
export function shortName(name: string): string {
  return name.endsWith('.service') ? name.slice(0, -'.service'.length) : name
}

/** Units in filter whose name or description contains query, any case. */
export function visible(units: readonly Unit[], filter: Filter, query: string): Unit[] {
  const q = query.trim().toLowerCase()
  return units.filter(
    (u) =>
      matches(u, filter) &&
      (!q || u.name.toLowerCase().includes(q) || u.description.toLowerCase().includes(q)),
  )
}

/** Which actions make sense in a state; the others are disabled. */
export function canDo(active: string, action: ServiceAction): boolean {
  const h = health(active)
  switch (action) {
    case 'start':
      return h === 'off' || h === 'failed'
    case 'stop':
      return h === 'ok' || h === 'busy'
  }
  return true
}

export type LineLevel = 'error' | 'warn' | null

const ERROR = /\b(error|err|failed|failure|fatal|crit(ical)?|emerg|alert|panic|segfault)\b/i
const WARN = /\b(warn(ing)?|deprecated)\b/i

/** Highlight level of a log line, from the words in it. */
export function lineLevel(line: string): LineLevel {
  if (ERROR.test(line)) return 'error'
  if (WARN.test(line)) return 'warn'
  return null
}
