// The lines a Logs window keeps, and what it shows of them.
import type { LogKind, LogSource } from '../../lib/api/logs'

/** Lines kept per window; older ones are dropped as new ones come. */
export const KEEP = 5000

/** lines with batch appended, keeping the last cap. */
export function append(lines: readonly string[], batch: readonly string[], cap = KEEP): string[] {
  const all = lines.concat(batch)
  return all.length > cap ? all.slice(all.length - cap) : all
}

/** Lines that contain query, any case; all of them for an empty query. */
export function filterLines(lines: readonly string[], query: string): readonly string[] {
  const q = query.trim().toLowerCase()
  return q ? lines.filter((l) => l.toLowerCase().includes(q)) : lines
}

/** Common log files offered when choosing a file. */
export const COMMON_FILES = [
  '/var/log/syslog',
  '/var/log/auth.log',
  '/var/log/kern.log',
  '/var/log/nginx/access.log',
  '/var/log/nginx/error.log',
  '/var/log/apache2/error.log',
  '/var/log/mysql/error.log',
  '/var/log/postgresql/postgresql-16-main.log',
] as const

/** Checks a target typed by the user for kind; returns why it is not usable. */
export function targetProblem(kind: LogKind, target: string): 'path' | 'unit' | 'name' | null {
  const v = target.trim()
  switch (kind) {
    case 'file':
      return v.startsWith('/') && !v.split('/').includes('..') ? null : 'path'
    case 'journal':
      return v === '' || /^[A-Za-z0-9][A-Za-z0-9:_.@\\-]*$/.test(v) ? null : 'unit'
    case 'docker':
      return /^[a-zA-Z0-9][a-zA-Z0-9_.-]*$/.test(v) ? null : 'name'
  }
}

/** The source as the backend wants it: units get ".service", paths are cleaned. */
export function normalize(src: LogSource): LogSource {
  const v = src.target.trim()
  if (src.kind === 'journal' && v && !v.endsWith('.service'))
    return { kind: 'journal', target: `${v}.service` }
  if (src.kind === 'file')
    return {
      kind: 'file',
      target: v
        .replace(/\/{2,}/g, '/')
        .replace(/(.)\/$/, '$1')
        .replace(/\/\.(?=\/|$)/g, ''),
    }
  return { kind: src.kind, target: v }
}

export function sameSource(a: LogSource | null, b: LogSource | null): boolean {
  return !!a && !!b && a.kind === b.kind && a.target === b.target
}
