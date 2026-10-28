import { describe, expect, it } from 'vitest'

import { type Candidate, intent, rank, score } from './search'

describe('score', () => {
  it('prefers prefixes, then word starts, then scattered letters', () => {
    const prefix = score('ngi', 'nginx.service')
    const word = score('ngi', 'my-nginx')
    const scattered = score('ngx', 'nginx')
    expect(prefix).toBeGreaterThan(word)
    expect(word).toBeGreaterThan(scattered)
    expect(scattered).toBeGreaterThan(0)
    expect(score('xyz', 'nginx')).toBe(0)
    expect(score('', 'anything')).toBe(1)
  })

  it('ignores case and favors shorter names', () => {
    expect(score('DOC', 'docker')).toBeGreaterThan(score('doc', 'docker-compose-thing'))
  })
})

describe('rank', () => {
  const c = (kind: Candidate['kind'], label: string, detail?: string): Candidate => ({
    kind,
    id: label,
    label,
    detail,
  })
  const all = [
    c('app', 'Terminal'),
    c('app', 'Monitor'),
    c('service', 'nginx.service', 'A high performance web server'),
    c('service', 'cron.service'),
    c('container', 'web', 'nginx:1.27'),
    c('folder', '/etc/nginx'),
  ]

  it('groups results, best first', () => {
    expect(rank(all, 'nginx').map((x) => `${x.kind}:${x.label}`)).toEqual([
      'service:nginx.service',
      'container:web',
      'folder:/etc/nginx',
    ])
  })

  it('shows apps first for an empty query', () => {
    expect(rank(all, '')[0]?.kind).toBe('app')
  })

  it('matches details only as a phrase', () => {
    const cron = c('service', 'cron.service', 'Regular background program processing daemon')
    expect(rank([cron], 'ngi')).toEqual([])
    expect(rank([cron], 'background')).toEqual([cron])
  })

  it('caps each group', () => {
    const many = Array.from({ length: 9 }, (_, i) => c('service', `svc${i}.service`))
    expect(rank(many, 'svc')).toHaveLength(5)
  })
})

describe('intent', () => {
  it.each([
    ['/etc/nginx/', { kind: 'path', path: '/etc/nginx' }],
    ['//var//log', { kind: 'path', path: '/var/log' }],
    ['/', { kind: 'path', path: '/' }],
    ['/etc/../root', null],
    ['> df -h', { kind: 'command', command: 'df -h' }],
    ['>', null],
    ['nginx', null],
  ])('%s', (q, want) => expect(intent(q)).toEqual(want))
})
