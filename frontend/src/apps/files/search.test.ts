import { describe as group, expect, it } from 'vitest'

import { contentAllowed, describe, highlight, mockSearch, modeOf } from './search'

group('modeOf', () => {
  it('maps scopes to server modes', () => {
    expect(modeOf('here')).toBeNull()
    expect(modeOf('tree')).toBe('name')
    expect(modeOf('content')).toBe('content')
  })
})

group('contentAllowed', () => {
  it.each([
    ['/', false],
    ['/proc', false],
    ['/proc/1', false],
    ['/dev/shm', false],
    ['/run', false],
    ['/running', true],
    ['/etc', true],
    ['/home/u/projects', true],
  ])('%s → %s', (dir, want) => expect(contentAllowed(dir)).toBe(want))
})

group('describe', () => {
  it('shows the folder relative to the start', () => {
    expect(describe({ path: '/etc/nginx/nginx.conf' }, '/etc')).toEqual({
      name: 'nginx.conf',
      where: 'nginx',
    })
    expect(describe({ path: '/etc/hosts' }, '/etc')).toEqual({ name: 'hosts', where: '.' })
    expect(describe({ path: '/etc/a/b/c' }, '/')).toEqual({ name: 'c', where: '/etc/a/b' })
    expect(describe({ path: '/etcetera/x' }, '/etc')).toEqual({ name: 'x', where: '/etcetera' })
  })
})

group('highlight', () => {
  it('splits around the first match', () => {
    expect(highlight('Error: Timeout here', 'timeout', false)).toEqual({
      before: 'Error: ',
      match: 'Timeout',
      after: ' here',
    })
    expect(highlight('Error: Timeout', 'timeout', true).match).toBe('')
  })
})

group('mockSearch', () => {
  it('finds names in the demo tree', () => {
    const r = mockSearch('/etc', 'conf', 'name', false)
    expect(r.hits.map((h) => h.path)).toContain('/etc/nginx/nginx.conf')
    expect(r.truncated).toBe(false)
  })

  it('caps at 500 hits', () => {
    const r = mockSearch('/usr', 'tool-*', 'name', false)
    expect(r.hits).toHaveLength(500)
    expect(r.truncated).toBe(true)
  })

  it('finds lines with their numbers', () => {
    const r = mockSearch('/etc/nginx', 'gzip', 'content', false)
    const hit = r.hits.find((h) => h.text === 'gzip on;')
    expect(hit?.path).toBe('/etc/nginx/nginx.conf')
    expect(hit?.line).toBe(20)
  })
})
