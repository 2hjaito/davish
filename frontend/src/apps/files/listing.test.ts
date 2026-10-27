import { describe, expect, it } from 'vitest'

import type { Entry } from '../../lib/api/files'
import {
  baseName,
  crumbs,
  filterEntries,
  formatModTime,
  joinPath,
  nextOrder,
  parentPath,
  scrollToRow,
  sortEntries,
  visibleRows,
  within,
} from './listing'

const entry = (name: string, dir: boolean, size = 0, modTime = 0): Entry => ({
  name,
  path: '/x/' + name,
  kind: dir ? 'dir' : 'file',
  dir,
  size,
  mode: '',
  perm: 0,
  modTime,
  uid: 0,
  gid: 0,
})

const names = (es: readonly Entry[]) => es.map((e) => e.name)

describe('sortEntries', () => {
  const list = [
    entry('file10', false, 5, 300),
    entry('b', true, 4096, 100),
    entry('File2', false, 50, 200),
    entry('a', true, 4096, 400),
    entry('.env', false, 50, 100),
  ]

  it('puts folders first and orders names naturally, ignoring case', () => {
    expect(names(sortEntries(list, { key: 'name', desc: false }))).toEqual([
      'a',
      'b',
      '.env',
      'File2',
      'file10',
    ])
  })

  it('reverses within each group, folders still first', () => {
    expect(names(sortEntries(list, { key: 'name', desc: true }))).toEqual([
      'b',
      'a',
      'file10',
      'File2',
      '.env',
    ])
  })

  it('sorts files by size with the name breaking ties; folders A to Z', () => {
    expect(names(sortEntries(list, { key: 'size', desc: true }))).toEqual([
      'a',
      'b',
      '.env',
      'File2',
      'file10',
    ])
  })

  it('sorts by modification time', () => {
    expect(names(sortEntries(list, { key: 'modified', desc: true }))).toEqual([
      'a',
      'b',
      'file10',
      'File2',
      '.env',
    ])
  })

  it('does not change its input', () => {
    const copy = [...list]
    sortEntries(list, { key: 'size', desc: false })
    expect(list).toEqual(copy)
  })
})

describe('nextOrder', () => {
  it('flips the current column', () => {
    expect(nextOrder({ key: 'name', desc: false }, 'name')).toEqual({ key: 'name', desc: true })
  })

  it('starts names ascending, sizes and dates descending', () => {
    expect(nextOrder({ key: 'size', desc: false }, 'name')).toEqual({ key: 'name', desc: false })
    expect(nextOrder({ key: 'name', desc: false }, 'size')).toEqual({ key: 'size', desc: true })
    expect(nextOrder({ key: 'name', desc: false }, 'modified')).toEqual({
      key: 'modified',
      desc: true,
    })
  })
})

describe('filterEntries', () => {
  const list = [
    entry('Nginx.conf', false),
    entry('sites-enabled', true),
    entry('mime.types', false),
  ]

  it('matches part of the name, ignoring case', () => {
    expect(names(filterEntries(list, ' NGINX '))).toEqual(['Nginx.conf'])
    expect(names(filterEntries(list, 'e'))).toEqual(['sites-enabled', 'mime.types'])
  })

  it('keeps everything for an empty query', () => {
    expect(filterEntries(list, '  ')).toBe(list)
  })
})

describe('paths', () => {
  it('joins without doubling the root slash', () => {
    expect(joinPath('/', 'etc')).toBe('/etc')
    expect(joinPath('/etc', 'nginx')).toBe('/etc/nginx')
  })

  it('finds the parent', () => {
    expect(parentPath('/etc/nginx')).toBe('/etc')
    expect(parentPath('/etc')).toBe('/')
    expect(parentPath('/')).toBe('/')
  })

  it('names the last part', () => {
    expect(baseName('/etc/nginx')).toBe('nginx')
    expect(baseName('/')).toBe('/')
  })

  it('splits into crumbs', () => {
    expect(crumbs('/')).toEqual([])
    expect(crumbs('/var/log')).toEqual([
      { name: 'var', path: '/var' },
      { name: 'log', path: '/var/log' },
    ])
  })

  it('tells whether a path is inside a place', () => {
    expect(within('/etc/nginx', '/etc')).toBe(true)
    expect(within('/etc', '/etc')).toBe(true)
    expect(within('/etcetera', '/etc')).toBe(false)
    expect(within('/anything', '/')).toBe(true)
  })
})

describe('formatModTime', () => {
  const words = { today: 'Hôm nay', yesterday: 'Hôm qua' }
  const now = new Date(2026, 9, 27, 10, 0)
  const at = (...args: [number, number, number, number, number]) =>
    new Date(...args).getTime() / 1000

  it('shows the time for today and yesterday', () => {
    expect(formatModTime(at(2026, 9, 27, 9, 5), now, words)).toBe('Hôm nay, 09:05')
    expect(formatModTime(at(2026, 9, 26, 23, 59), now, words)).toBe('Hôm qua, 23:59')
  })

  it('shows the date for anything older', () => {
    expect(formatModTime(at(2026, 9, 25, 12, 0), now, words)).toBe('25/10/2026')
    expect(formatModTime(at(2025, 0, 3, 12, 0), now, words)).toBe('03/01/2025')
  })
})

describe('visibleRows', () => {
  it('covers the viewport plus overscan, clamped to the list', () => {
    expect(visibleRows(0, 300, 30, 1000, 5)).toEqual({ start: 0, end: 15 })
    expect(visibleRows(3000, 300, 30, 1000, 5)).toEqual({ start: 95, end: 115 })
    expect(visibleRows(29_800, 300, 30, 1000, 5)).toEqual({ start: 988, end: 1000 })
    expect(visibleRows(0, 300, 30, 3, 5)).toEqual({ start: 0, end: 3 })
  })
})

describe('scrollToRow', () => {
  it('scrolls only when the row is out of view', () => {
    expect(scrollToRow(5, 0, 300, 30)).toBeNull()
    expect(scrollToRow(12, 0, 300, 30)).toBe(90)
    expect(scrollToRow(2, 120, 300, 30)).toBe(60)
  })
})
