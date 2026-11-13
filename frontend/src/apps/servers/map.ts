// The server map: which requests reach which program. Pure, so it is tested
// on its own; the window reads nginx sites, listening ports and containers
// and hands them here (docs/PLAN.md M45).
import type { Container } from '../../lib/api/docker'
import type { NginxSite } from '../../lib/api/nginx'
import type { Port } from '../../lib/api/ports'

/** Where nginx sends a request, from a proxy_pass. */
export type Target =
  | { kind: 'port'; host: string; port: number }
  | { kind: 'socket'; path: string }
  /** An upstream block or a variable: its servers are not read. */
  | { kind: 'name'; name: string }

export type Warning = 'noListener' | 'dbExposed' | 'certExpiring'

/** One way in: through an nginx site, or straight to a port. */
export interface Route {
  key: string
  /** The site's first server_name; undefined for a direct route. */
  domain?: string
  /** More server_names of the same block. */
  aliases: number
  /** The site's file name, to open in Nginx. */
  site?: string
  /** nginx's listening ports for the block, TLS ones first. */
  listen: number[]
  tls: boolean
  /** Where nginx passes it; undefined for a static site or a direct route. */
  target?: Target
  /** A static site's root folder. */
  root?: string
  /** The socket that answers, when one listens on the target's port. */
  listener?: Port
  /** The container behind it, by docker-proxy or by name on a docker network. */
  container?: Container
  warnings: Warning[]
}

/** Database ports that should not face the Internet. */
export const DB_PORTS: ReadonlySet<number> = new Set([
  3306, 5432, 6379, 27017, 9200, 11211, 5984, 9042,
])
const DB_PROCESS = /^(mysqld|mariadbd|postgres|redis-server|mongod|memcached|couchdb)$/

/** A certificate this close to its end is flagged, as Nginx does. */
const CERT_WARN_S = 14 * 86400

const LOCAL = new Set(['127.0.0.1', 'localhost', '0.0.0.0', '::1', '[::1]', '::', '[::]'])

/** "http://127.0.0.1:3000/api", "http://unix:/run/a.sock:", "http://backend" → a Target. */
export function parseProxyPass(pass: string): Target {
  const m = /^(https?):\/\/(.*)$/i.exec(pass.trim())
  if (!m) return { kind: 'name', name: pass.trim() }
  const scheme = (m[1] ?? 'http').toLowerCase()
  const rest = m[2] ?? ''
  if (rest.startsWith('unix:')) {
    const path = rest.slice('unix:'.length).replace(/:.*$/, '')
    return { kind: 'socket', path }
  }
  const hostPort = rest.split('/')[0] ?? ''
  if (hostPort.includes('$')) return { kind: 'name', name: hostPort }
  const hp = /^(\[[^\]]+\]|[^:]+)(?::(\d+))?$/.exec(hostPort)
  if (!hp) return { kind: 'name', name: hostPort }
  const host = (hp[1] ?? '').toLowerCase()
  if (hp[2]) return { kind: 'port', host, port: Number(hp[2]) }
  // No port: a named upstream, or the scheme's port on a host.
  if (!LOCAL.has(host) && !host.includes('.')) return { kind: 'name', name: host }
  return { kind: 'port', host, port: scheme === 'https' ? 443 : 80 }
}

/** "443 ssl http2", "[::]:80", "127.0.0.1:8080" → 443, 80, 8080. */
export function listenPort(listen: string): number | undefined {
  const first = listen.trim().split(/\s+/)[0] ?? ''
  const m = /(?:^|:)(\d+)$/.exec(first)
  return m ? Number(m[1]) : undefined
}

function findContainer(list: readonly Container[], name: string): Container | undefined {
  const n = name.toLowerCase()
  return (
    list.find((c) => c.name.toLowerCase() === n) ?? list.find((c) => c.service?.toLowerCase() === n)
  )
}

function listenerFor(ports: readonly Port[], port: number, host: string): Port | undefined {
  const local = LOCAL.has(host)
  return ports.find((p) => p.port === port && (local || p.addr === host || !p.loopback))
}

export interface MapInput {
  sites: readonly NginxSite[]
  ports: readonly Port[]
  containers: readonly Container[]
  /** Unix seconds. */
  now: number
}

export function routes({ sites, ports, containers, now }: MapInput): Route[] {
  const out: Route[] = []
  const proxied = new Set<number>()
  const nginxPorts = new Set<number>()

  for (const site of sites) {
    if (!site.enabled) continue
    site.servers.forEach((srv, si) => {
      const listen = [...new Set(srv.listen.map(listenPort).filter((p) => p !== undefined))].sort(
        (a, b) => Number(b === 443) - Number(a === 443) || a - b,
      )
      for (const p of listen) nginxPorts.add(p)
      const names = srv.names.filter((n) => n && n !== '_' && n !== 'localhost')
      const certs = site.certs.filter((c) => srv.certs.includes(c.path))
      const expiring = certs.some((c) => c.notAfter > 0 && c.notAfter - now < CERT_WARN_S)
      const base = {
        domain: names[0],
        aliases: Math.max(0, names.length - 1),
        site: site.name,
        listen,
        tls: srv.ssl,
      }
      const passes = [...new Set(srv.proxyPass)]
      if (passes.length === 0) {
        out.push({
          ...base,
          key: `${site.name}#${si}`,
          root: srv.root,
          warnings: expiring ? ['certExpiring'] : [],
        })
        return
      }
      passes.forEach((pass, pi) => {
        const target = parseProxyPass(pass)
        const r: Route = { ...base, key: `${site.name}#${si}#${pi}`, target, warnings: [] }
        if (target.kind === 'port') {
          proxied.add(target.port)
          if (!LOCAL.has(target.host) && !target.host.includes('.')) {
            // nginx in Docker reaching a service by name on its network.
            r.container = findContainer(containers, target.host)
          }
          if (!r.container) {
            r.listener = listenerFor(ports, target.port, target.host)
            if (r.listener?.container) r.container = findContainer(containers, r.listener.container)
            if (!r.listener && LOCAL.has(target.host)) r.warnings.push('noListener')
          }
        } else if (target.kind === 'name') {
          r.container = findContainer(containers, target.name)
        }
        if (expiring) r.warnings.push('certExpiring')
        out.push(r)
      })
    })
  }

  // Ports reachable from outside that no site proxies to and nginx does not own.
  const direct = new Map<number, Port>()
  for (const p of ports)
    if (!p.loopback && p.exposed !== 'no' && !nginxPorts.has(p.port) && !proxied.has(p.port))
      if (!direct.has(p.port)) direct.set(p.port, p)
  for (const p of [...direct.values()].sort((a, b) => a.port - b.port)) {
    const container = p.container ? findContainer(containers, p.container) : undefined
    const db = DB_PORTS.has(p.port) || DB_PROCESS.test(p.process ?? '')
    out.push({
      key: `port:${p.port}`,
      aliases: 0,
      listen: [],
      tls: false,
      listener: p,
      container,
      warnings: db && p.exposed === 'yes' ? ['dbExposed'] : [],
    })
  }
  // A block that neither proxies nor serves files is the usual http → https
  // redirect: left out when its domain has a real route.
  const served = new Set(out.filter((r) => r.target || r.root).map((r) => r.domain))
  return out.filter((r) => r.target || r.root || !r.domain || !served.has(r.domain) || r.listener)
}

/** Routes with a warning first, then sites by domain, then direct ports. */
export function sortRoutes(list: readonly Route[]): Route[] {
  const rank = (r: Route) => (r.warnings.length ? 0 : r.domain || r.site ? 1 : 2)
  return [...list].sort(
    (a, b) =>
      rank(a) - rank(b) ||
      // Sites by domain, then direct ports, which have none.
      (a.domain ?? '\uffff').localeCompare(b.domain ?? '\uffff') ||
      (a.listener?.port ?? 0) - (b.listener?.port ?? 0),
  )
}

/** Programs that listen without a container: sshd, a node app run by pm2… */
export function processes(ports: readonly Port[]): string[] {
  return [...new Set(ports.flatMap((p) => (p.process && !p.container ? [p.process] : [])))].sort()
}
