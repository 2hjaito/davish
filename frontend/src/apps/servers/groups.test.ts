import { describe, expect, it } from 'vitest'

import {
  activeFilter,
  groupNames,
  matches,
  sections,
  stateOf,
  tagNames,
  ungroupedCount,
  type Listed,
} from './groups'

const s = (group: string | undefined, tag?: string, state = 'connected'): Listed => ({
  group,
  tag,
  state,
})

describe('stateOf', () => {
  it('buckets states, counting pending connections as not connected', () => {
    expect(stateOf('connected')).toBe('connected')
    expect(stateOf('failed')).toBe('failed')
    expect(stateOf('disconnected')).toBe('disconnected')
    expect(stateOf('connecting')).toBe('disconnected')
    expect(stateOf('reconnecting')).toBe('disconnected')
  })
})

describe('groupNames and tagNames', () => {
  it('count per name in order of first mention, ignoring case and blanks', () => {
    const list = [s('Prod app'), s('staging'), s('prod app'), s(' '), s(undefined)]
    expect(groupNames(list)).toEqual([
      { name: 'Prod app', count: 2 },
      { name: 'staging', count: 1 },
    ])
    expect(ungroupedCount(list)).toBe(2)
  })

  it('read tags the same way', () => {
    expect(tagNames([s('a', 'prod'), s('b', 'Prod'), s('c')])).toEqual([{ name: 'prod', count: 2 }])
  })
})

describe('matches', () => {
  it('keeps everything for all', () => {
    expect(matches(s(undefined), 'all')).toBe(true)
  })

  it('filters by connection state', () => {
    expect(matches(s('a', undefined, 'failed'), 'failed')).toBe(true)
    expect(matches(s('a', undefined, 'connecting'), 'disconnected')).toBe(true)
    expect(matches(s('a'), 'failed')).toBe(false)
  })

  it('filters by group, ignoring case, and by ungrouped', () => {
    expect(matches(s('Prod'), 'group:prod')).toBe(true)
    expect(matches(s('staging'), 'group:prod')).toBe(false)
    expect(matches(s(undefined), 'ungrouped')).toBe(true)
    expect(matches(s('prod'), 'ungrouped')).toBe(false)
    expect(matches(s(undefined), 'group:prod')).toBe(false)
  })

  it('filters by tag', () => {
    expect(matches(s('a', 'Prod'), 'tag:prod')).toBe(true)
    expect(matches(s('a'), 'tag:prod')).toBe(false)
  })
})

describe('activeFilter', () => {
  const list = [s('prod', 'prod'), s(undefined)]

  it('keeps a filter that still matches something', () => {
    expect(activeFilter(list, 'group:prod')).toBe('group:prod')
    expect(activeFilter(list, 'tag:prod')).toBe('tag:prod')
    expect(activeFilter(list, 'ungrouped')).toBe('ungrouped')
    expect(activeFilter(list, 'failed')).toBe('failed')
  })

  it('falls back to all when the group or tag is gone', () => {
    expect(activeFilter(list, 'group:staging')).toBe('all')
    expect(activeFilter(list, 'tag:qa')).toBe('all')
    expect(activeFilter([s('prod')], 'ungrouped')).toBe('all')
  })

  it('falls back to all for an unknown id', () => {
    expect(activeFilter(list, 'nonsense')).toBe('all')
  })
})

describe('sections', () => {
  it('is one nameless section when no server has a group', () => {
    const list = [s(undefined), s(' ')]
    expect(sections(list)).toEqual([{ name: '', servers: list }])
  })

  it('keeps the order each group first appears, ungrouped last', () => {
    const a = s('staging')
    const b = s(undefined)
    const c = s('prod')
    const d = s('Staging')
    const out = sections([a, b, c, d])
    expect(out.map((x) => x.name)).toEqual(['staging', 'prod', ''])
    expect(out[0]?.servers).toEqual([a, d])
    expect(out[2]?.servers).toEqual([b])
  })

  it('follows the given order so folders stay put when servers are re-sorted', () => {
    const list = [s('b'), s('a'), s('c')]
    expect(sections(list, ['a', 'b', 'c']).map((x) => x.name)).toEqual(['a', 'b', 'c'])
    expect(sections(list, ['C']).map((x) => x.name)).toEqual(['c', 'b', 'a'])
  })

  it('has no nameless section when every server is grouped', () => {
    expect(sections([s('a'), s('b')]).map((x) => x.name)).toEqual(['a', 'b'])
  })

  it('is empty-safe', () => {
    expect(sections([])).toEqual([{ name: '', servers: [] }])
  })
})
