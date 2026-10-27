import { expect, it } from 'vitest'

import type { NginxSite } from '../../lib/api/nginx'
import { certState, daysLeft, mainName, visible, worstCert } from './sites'

const site = (name: string, enabled: boolean, names: string[], notAfter?: number): NginxSite => ({
  name,
  path: `/etc/nginx/sites-available/${name}`,
  kind: 'site',
  enabled,
  toggleable: true,
  servers: [{ names, listen: ['80'], ssl: false, certs: [], proxyPass: [], line: 1 }],
  upstreams: [],
  certs: notAfter === undefined ? [] : [{ path: '/c.pem', notAfter, names: [] }],
})

const now = Date.UTC(2026, 9, 27)
const day = 86_400

it('filters and searches server names', () => {
  const list = [
    site('default', true, ['_']),
    site('shop', true, ['shop.example.com']),
    site('old', false, ['blog.test']),
  ]
  expect(visible(list, 'enabled', '').map((s) => s.name)).toEqual(['default', 'shop'])
  expect(visible(list, 'all', 'BLOG').map((s) => s.name)).toEqual(['old'])
  expect(visible(list, 'disabled', 'shop')).toEqual([])
})

it('names a site by its first real server name', () => {
  expect(mainName(site('a', true, ['_', 'x.test']))).toBe('x.test')
  expect(mainName(site('b', true, ['_']))).toBe('80')
})

it('grades certificates', () => {
  const at = (d: number) => now / 1000 + d * day
  expect(daysLeft({ path: '', names: [], notAfter: at(9.5) }, now)).toBe(9)
  expect(certState({ path: '', names: [], notAfter: at(30) }, now)).toBe('ok')
  expect(certState({ path: '', names: [], notAfter: at(14) }, now)).toBe('soon')
  expect(certState({ path: '', names: [], notAfter: at(-1) }, now)).toBe('expired')
  expect(certState({ path: '', names: [], notAfter: 0, error: 'unreadable' }, now)).toBe('unknown')
  expect(worstCert(site('s', true, [], at(3)), now)).toBe('soon')
  expect(worstCert(site('s', true, []), now)).toBeNull()
})
