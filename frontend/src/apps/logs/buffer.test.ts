import { describe, expect, it } from 'vitest'

import { append, filterLines, normalize, sameSource, targetProblem } from './buffer'

describe('append', () => {
  it('keeps the newest lines up to the cap', () => {
    expect(append(['a', 'b'], ['c'], 5)).toEqual(['a', 'b', 'c'])
    expect(append(['a', 'b', 'c'], ['d', 'e', 'f'], 4)).toEqual(['c', 'd', 'e', 'f'])
    expect(append([], ['1', '2', '3', '4', '5', '6'], 2)).toEqual(['5', '6'])
  })
})

it('filters any case and keeps everything for blanks', () => {
  const lines = ['GET / 200', 'ERROR db down', 'error again']
  expect(filterLines(lines, 'Error')).toEqual(['ERROR db down', 'error again'])
  expect(filterLines(lines, '  ')).toBe(lines)
})

describe('targets', () => {
  it.each([
    ['file', '/var/log/syslog', null],
    ['file', 'var/log/syslog', 'path'],
    ['file', '/var/log/../../etc/shadow', 'path'],
    ['journal', '', null],
    ['journal', 'nginx', null],
    ['journal', 'getty@tty1.service', null],
    ['journal', '-f', 'unit'],
    ['docker', 'web_1', null],
    ['docker', '--help', 'name'],
    ['docker', '', 'name'],
  ] as const)('%s %s', (kind, target, want) => expect(targetProblem(kind, target)).toBe(want))

  it('normalizes what the user typed', () => {
    expect(normalize({ kind: 'journal', target: ' nginx ' })).toEqual({
      kind: 'journal',
      target: 'nginx.service',
    })
    expect(normalize({ kind: 'journal', target: '' })).toEqual({ kind: 'journal', target: '' })
    expect(normalize({ kind: 'file', target: '/var//log/./nginx/' })).toEqual({
      kind: 'file',
      target: '/var/log/nginx',
    })
    expect(normalize({ kind: 'docker', target: ' web ' })).toEqual({
      kind: 'docker',
      target: 'web',
    })
  })

  it('compares sources', () => {
    expect(sameSource({ kind: 'file', target: '/a' }, { kind: 'file', target: '/a' })).toBe(true)
    expect(sameSource({ kind: 'file', target: '/a' }, { kind: 'docker', target: '/a' })).toBe(false)
    expect(sameSource(null, null)).toBe(false)
  })
})
