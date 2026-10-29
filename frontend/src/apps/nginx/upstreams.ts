// Addresses a reverse proxy can point at: what listens on the server.
import type { Port } from '../../lib/api/ports'
import { backend as ports } from '../ports/backend'

export interface Choice {
  value: string
  label: string
}

// nginx itself and sshd are never what a site proxies to.
const skip = new Set(['nginx', 'sshd'])

export function choices(list: readonly Port[]): Choice[] {
  const seen = new Set<string>()
  const out: Choice[] = []
  for (const p of list) {
    if (p.process && skip.has(p.process)) continue
    const host =
      p.loopback || p.addr === '0.0.0.0' || p.addr === '::' || p.addr === '*' ? '127.0.0.1' : p.addr
    const value = `${host.includes(':') ? `[${host}]` : host}:${p.port}`
    if (seen.has(value)) continue
    seen.add(value)
    out.push({ value, label: p.container ?? p.process ?? '' })
  }
  return out.sort((a, b) => a.value.localeCompare(b.value, undefined, { numeric: true }))
}

/** What listens on the server; empty when it cannot be read. */
export const upstreamChoices = (serverID: string): Promise<Choice[]> =>
  ports
    .list(serverID)
    .then((l) => choices(l.ports))
    .catch(() => [])
