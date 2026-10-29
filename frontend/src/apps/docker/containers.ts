// Pure helpers of the Docker app: filters, states and times.
import type { Container, ContainerAction, Image } from '../../lib/api/docker'
import { t, type AgoUnit } from '../../lib/i18n'

export type Filter = 'all' | 'running' | 'restarting' | 'stopped'

export const FILTERS: readonly Filter[] = ['all', 'running', 'restarting', 'stopped']

/** The image side of the sidebar. */
export type ImageFilter = 'images' | 'unused'

export const IMAGE_FILTERS: readonly ImageFilter[] = ['images', 'unused']

/** Images in filter whose name or ID contains query; unused ones have no container. */
export function visibleImages(list: readonly Image[], filter: ImageFilter, query: string): Image[] {
  const q = query.trim().toLowerCase()
  return list.filter(
    (i) =>
      (filter === 'images' || i.containers === 0) &&
      (!q || `${i.repository}:${i.tag} ${i.id}`.toLowerCase().includes(q)),
  )
}

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

/** Docker's "6 days ago", "About an hour ago" in the UI language; the text as is otherwise. */
export function ago(text: string): string {
  const s = text.trim()
  if (/^less than a second/i.test(s)) return t.docker.justNow
  const m = /^(about an?|an?|\d+)\s+(second|minute|hour|day|week|month|year)s?\s+ago$/i.exec(s)
  if (!m) return s
  const n = /^\d+$/.test(m[1] ?? '') ? Number(m[1]) : 1
  const unit = (m[2] ?? '').toLowerCase() as AgoUnit
  return t.docker.ago(n, unit, /^about/i.test(m[1] ?? ''))
}

/** One published or exposed port of a container. */
export interface PortChip {
  /** The port on the server; undefined when only exposed inside Docker. */
  host?: number
  container: string
  /** Published on 127.0.0.1 only. */
  local: boolean
}

/** Docker's "0.0.0.0:8080->80/tcp, :::8080->80/tcp, 5432/tcp", once per port. */
export function parsePorts(ports: string): PortChip[] {
  const out: PortChip[] = []
  const seen = new Set<string>()
  for (const raw of ports.split(',')) {
    const p = raw.trim()
    if (!p) continue
    const m = /^(.*):(\d+)->(\S+)$/.exec(p)
    const chip: PortChip = m
      ? {
          host: Number(m[2]),
          container: (m[3] ?? '').replace(/\/tcp$/, ''),
          local: m[1] === '127.0.0.1',
        }
      : { container: p.replace(/\/tcp$/, ''), local: false }
    const key = `${chip.host ?? ''}:${chip.container}`
    if (seen.has(key)) continue
    seen.add(key)
    out.push(chip)
  }
  return out
}

/** A compose project's containers, or one container on its own. */
export interface Group {
  key: string
  /** The compose project; undefined for a lone container. */
  project?: string
  containers: Container[]
}

/** Compose projects first by name, each with its containers; lone ones after. */
export function groups(list: readonly Container[]): Group[] {
  const byProject = new Map<string, Container[]>()
  const lone: Group[] = []
  for (const c of list) {
    if (!c.project) {
      lone.push({ key: c.name, containers: [c] })
      continue
    }
    byProject.set(c.project, [...(byProject.get(c.project) ?? []), c])
  }
  const projects = [...byProject.entries()]
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([project, containers]) => ({ key: `project:${project}`, project, containers }))
  return [...projects, ...lone]
}
