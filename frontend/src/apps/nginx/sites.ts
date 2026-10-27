// Pure helpers of the Nginx app.
import type { NginxCert, NginxSite } from '../../lib/api/nginx'

export type Filter = 'all' | 'enabled' | 'disabled'

export const FILTERS: readonly Filter[] = ['all', 'enabled', 'disabled']

export function matches(s: NginxSite, f: Filter): boolean {
  if (f === 'enabled') return s.enabled
  if (f === 'disabled') return !s.enabled
  return true
}

/** A stable key: the same name can sit in sites-available and conf.d. */
export const siteKey = (s: NginxSite) => s.path

/** Sites in filter whose name, server names or file contain query. */
export function visible(list: readonly NginxSite[], f: Filter, query: string): NginxSite[] {
  const q = query.trim().toLowerCase()
  return list.filter(
    (s) =>
      matches(s, f) &&
      (!q ||
        s.name.toLowerCase().includes(q) ||
        s.path.toLowerCase().includes(q) ||
        s.servers.some((v) => v.names.some((n) => n.toLowerCase().includes(q)))),
  )
}

/** The first real server name, for the list: "_" and empty names say nothing. */
export function mainName(s: NginxSite): string {
  for (const v of s.servers) for (const n of v.names) if (n && n !== '_') return n
  return s.servers[0]?.listen[0] ?? ''
}

export type CertState = 'ok' | 'soon' | 'expired' | 'unknown'

/** Whole days left, rounded down; negative once expired. */
export function daysLeft(c: NginxCert, now = Date.now()): number {
  return Math.floor((c.notAfter * 1000 - now) / 86_400_000)
}

export function certState(c: NginxCert, now = Date.now()): CertState {
  if (!c.notAfter) return 'unknown'
  const d = daysLeft(c, now)
  if (d < 0) return 'expired'
  return d <= 14 ? 'soon' : 'ok'
}

/** The worst state among a site's certificates, for the list dot. */
export function worstCert(s: NginxSite, now = Date.now()): CertState | null {
  const order: CertState[] = ['expired', 'soon', 'unknown', 'ok']
  const states = s.certs.map((c) => certState(c, now))
  return order.find((o) => states.includes(o)) ?? null
}
