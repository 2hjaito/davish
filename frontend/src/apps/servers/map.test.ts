import { describe, expect, it } from 'vitest'

import type { Container } from '../../lib/api/docker'
import type { NginxServer, NginxSite } from '../../lib/api/nginx'
import type { Port } from '../../lib/api/ports'
import { listenPort, parseProxyPass, processes, routes, sortRoutes } from './map'

const NOW = 1_800_000_000

const port = (p: number, extra: Partial<Port> = {}): Port => ({
  addr: '0.0.0.0',
  port: p,
  loopback: false,
  exposed: 'yes',
  ...extra,
})

const container = (name: string, project?: string, service?: string): Container => ({
  id: name,
  name,
  image: 'x',
  state: 'running',
  status: '',
  ports: '',
  command: '',
  created: '',
  runningFor: '',
  project,
  service,
})

const server = (s: Partial<NginxServer>): NginxServer => ({
  names: [],
  listen: ['80'],
  ssl: false,
  certs: [],
  proxyPass: [],
  line: 1,
  ...s,
})

const site = (name: string, servers: NginxServer[], extra: Partial<NginxSite> = {}): NginxSite => ({
  name,
  path: `/etc/nginx/sites-available/${name}`,
  kind: 'site',
  enabled: true,
  toggleable: true,
  servers,
  upstreams: [],
  certs: [],
  ...extra,
})

describe('proxy_pass', () => {
  it('reads hosts, ports, sockets and names', () => {
    expect(
      [
        'http://127.0.0.1:3000',
        'http://localhost:8080/api/',
        'https://localhost',
        'http://[::1]:9000',
        'http://unix:/run/app.sock:/',
        'http://backend',
        'http://web:3000',
        'http://$upstream',
        'http://example.com',
      ].map(parseProxyPass),
    ).toEqual([
      { kind: 'port', host: '127.0.0.1', port: 3000 },
      { kind: 'port', host: 'localhost', port: 8080 },
      { kind: 'port', host: 'localhost', port: 443 },
      { kind: 'port', host: '[::1]', port: 9000 },
      { kind: 'socket', path: '/run/app.sock' },
      { kind: 'name', name: 'backend' },
      { kind: 'port', host: 'web', port: 3000 },
      { kind: 'name', name: '$upstream' },
      { kind: 'port', host: 'example.com', port: 80 },
    ])
  })

  it('reads listen lines', () => {
    expect(['443 ssl http2', '[::]:80', '127.0.0.1:8080', 'unix:/x.sock'].map(listenPort)).toEqual([
      443,
      80,
      8080,
      undefined,
    ])
  })
})

describe('routes', () => {
  const containers = [container('shop-web-1', 'shop', 'web'), container('shop-db-1', 'shop', 'db')]
  const ports = [
    port(22, { process: 'sshd' }),
    port(80, { process: 'nginx' }),
    port(443, { process: 'nginx' }),
    port(8080, {
      addr: '127.0.0.1',
      loopback: true,
      process: 'docker-proxy',
      container: 'shop-web-1',
    }),
    port(5432, { process: 'docker-proxy', container: 'shop-db-1' }),
    port(3001, { addr: '127.0.0.1', loopback: true, process: 'node' }),
  ]
  const sites = [
    site(
      'shop',
      [
        server({
          names: ['shop.test', 'www.shop.test'],
          listen: ['80', '443 ssl'],
          ssl: true,
          certs: ['/etc/le/shop.pem'],
          proxyPass: ['http://127.0.0.1:8080'],
        }),
      ],
      { certs: [{ path: '/etc/le/shop.pem', notAfter: NOW + 5 * 86400, names: ['shop.test'] }] },
    ),
    site('blog', [server({ names: ['blog.test'], proxyPass: ['http://localhost:4000'] })]),
    site('docs', [server({ names: ['docs.test'], root: '/var/www/docs' })]),
    site('api', [server({ names: ['api.test'], proxyPass: ['http://localhost:3001'] })]),
    site('off', [server({ names: ['off.test'], proxyPass: ['http://localhost:1'] })], {
      enabled: false,
    }),
  ]
  const got = routes({ sites, ports, containers, now: NOW })
  const by = (k: string) => got.find((r) => r.key.startsWith(k))

  it('follows a site to its container', () => {
    const r = by('shop#')
    expect(r).toMatchObject({ domain: 'shop.test', aliases: 1, tls: true, listen: [443, 80] })
    expect(r?.listener?.port).toBe(8080)
    expect(r?.container?.name).toBe('shop-web-1')
    expect(r?.warnings).toEqual(['certExpiring'])
  })

  it('follows a site to a plain process', () => {
    expect(by('api#')?.listener?.process).toBe('node')
    expect(by('api#')?.container).toBeUndefined()
  })

  it('flags an upstream nobody listens on', () => {
    expect(by('blog#')?.warnings).toEqual(['noListener'])
  })

  it('drops the http → https redirect block of a served domain', () => {
    const r = routes({
      sites: [
        site('shop', [
          server({ names: ['shop.test'], listen: ['80'] }),
          server({
            names: ['shop.test'],
            listen: ['443 ssl'],
            ssl: true,
            proxyPass: ['http://localhost:3000'],
          }),
          server({ names: ['lone.test'], listen: ['80'] }),
        ]),
      ],
      ports: [port(3000, { loopback: true, addr: '127.0.0.1', process: 'node' })],
      containers: [],
      now: NOW,
    })
    expect(r.map((x) => [x.domain, x.listen])).toEqual([
      ['shop.test', [443]],
      ['lone.test', [80]],
    ])
  })

  it('keeps static sites and skips disabled ones', () => {
    expect(by('docs#')?.root).toBe('/var/www/docs')
    expect(by('docs#')?.target).toBeUndefined()
    expect(by('off#')).toBeUndefined()
  })

  it('lists ports open without a proxy, flagging databases', () => {
    const direct = got.filter((r) => r.key.startsWith('port:'))
    // 80/443 are nginx's, 8080 and 3001 are loopback.
    expect(direct.map((r) => r.listener?.port)).toEqual([22, 5432])
    expect(direct[1]?.container?.name).toBe('shop-db-1')
    expect(direct[1]?.warnings).toEqual(['dbExposed'])
    expect(direct[0]?.warnings).toEqual([])
  })

  it('does not flag a database the firewall may block', () => {
    const r = routes({
      sites: [],
      ports: [port(3306, { exposed: 'unknown', process: 'mysqld' })],
      containers: [],
      now: NOW,
    })
    expect(r[0]?.warnings).toEqual([])
  })

  it('finds a container nginx reaches by name on a docker network', () => {
    const r = routes({
      sites: [site('app', [server({ names: ['app.test'], proxyPass: ['http://web:3000'] })])],
      ports: [],
      containers,
      now: NOW,
    })
    expect(r[0]?.container?.name).toBe('shop-web-1')
    expect(r[0]?.warnings).toEqual([])
  })

  it('puts warnings first', () => {
    expect(sortRoutes(got).map((r) => r.key.split('#')[0])).toEqual([
      'blog',
      'shop',
      'port:5432',
      'api',
      'docs',
      'port:22',
    ])
  })

  it('names the listening programs outside Docker', () => {
    expect(processes(ports)).toEqual(['nginx', 'node', 'sshd'])
  })
})
