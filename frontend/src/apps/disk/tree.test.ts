import { describe, expect, it } from 'vitest'

import {
  buildTree,
  knownChildren,
  matchingRows,
  nameOf,
  parentOf,
  visibleRows,
  withoutPath,
} from './tree'

const dirs = [
  { path: '/var', size: 100 },
  { path: '/var/lib', size: 70 },
  { path: '/var/log', size: 20 },
  { path: '/var/lib/docker', size: 60 },
  { path: '/home', size: 30 },
]

describe('tree', () => {
  it('splits paths', () => {
    expect(parentOf('/var/lib')).toBe('/var')
    expect(parentOf('/var')).toBe('/')
    expect(nameOf('/var/lib')).toBe('lib')
  })

  it('shows the root children, then expanded folders, largest first', () => {
    const tree = buildTree(dirs, {})
    const rows = visibleRows('/', 130, tree, {}, new Set(['/var']))
    expect(rows.map((r) => [r.path, r.depth, r.parentSize])).toEqual([
      ['/var', 0, 130],
      ['/var/lib', 1, 100],
      ['/var/log', 1, 100],
      ['/home', 0, 130],
    ])
    expect(rows[0]?.hasChildren).toBe(true)
    expect(rows[2]?.hasChildren).toBe(false)
  })

  it('knows children within the overview depth or once measured', () => {
    expect(knownChildren('/', '/var/lib', {})).toBe(true)
    expect(knownChildren('/', '/var/lib/docker', {})).toBe(false)
    const measured = {
      '/var/lib/docker': {
        dir: '/var/lib/docker',
        size: 60,
        partial: false,
        entries: [{ path: '/var/lib/docker/overlay2', size: 55 }],
      },
    }
    expect(knownChildren('/', '/var/lib/docker', measured)).toBe(true)
    const tree = buildTree(dirs, measured)
    const rows = visibleRows(
      '/',
      130,
      tree,
      measured,
      new Set(['/var', '/var/lib', '/var/lib/docker']),
    )
    expect(rows.map((r) => r.path)).toContain('/var/lib/docker/overlay2')
  })

  it('counts depth from a mount that is not /', () => {
    expect(knownChildren('/srv', '/srv/a/b', {})).toBe(true)
    expect(knownChildren('/srv', '/srv/a/b/c', {})).toBe(false)
  })

  it('finds folders by name for the search box', () => {
    const rows = matchingRows('/', buildTree(dirs, {}), 'LI')
    expect(rows.map((r) => r.path)).toEqual(['/var/lib'])
  })
})

describe('withoutPath', () => {
  it('drops the path and takes its size off the folders above and the disk', () => {
    const o = {
      mount: { path: '/', total: 200, used: 150 },
      measured: 130,
      dirs,
      files: [{ path: '/var/lib/docker/x.log', size: 40 }],
      unreadable: [],
      deleted: [],
      unmeasured: 20,
      root: false,
      partial: false,
      sudoMeasure: '',
      askSudo: false,
    }
    const got = withoutPath(o, '/var/lib/docker', 60)
    expect(got.mount.used).toBe(90)
    expect(got.measured).toBe(70)
    expect(got.files).toEqual([])
    expect(got.dirs).toEqual([
      { path: '/var', size: 40 },
      { path: '/var/lib', size: 10 },
      { path: '/var/log', size: 20 },
      { path: '/home', size: 30 },
    ])
  })
})
