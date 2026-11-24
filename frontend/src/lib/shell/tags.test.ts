import { describe, expect, it } from 'vitest'

import { activeFilter, matchesTag, tagGroups } from './tags'

describe('tagGroups', () => {
  it('counts servers per tag and ignores untagged ones', () => {
    const groups = tagGroups([
      { tag: 'prod', color: '#d6455a' },
      { tag: 'prod' },
      { tag: 'staging', color: '#e08a12' },
      {},
      { tag: '  ' },
    ])
    expect(groups.map((g) => [g.tag, g.count])).toEqual([
      ['staging', 1],
      ['prod', 2],
    ])
  })

  it('treats different capitalization as one tag and keeps the first spelling', () => {
    const groups = tagGroups([{ tag: 'Prod' }, { tag: 'prod' }])
    expect(groups).toHaveLength(1)
    expect(groups[0]).toMatchObject({ tag: 'Prod', count: 2 })
  })

  it('takes the color from the first server that has one', () => {
    const groups = tagGroups([{ tag: 'prod' }, { tag: 'prod', color: '#d6455a' }])
    expect(groups[0]?.color).toBe('#d6455a')
  })

  it('lists the presets first, then custom tags alphabetically', () => {
    const groups = tagGroups([{ tag: 'zeta' }, { tag: 'prod' }, { tag: 'alpha' }, { tag: 'local' }])
    expect(groups.map((g) => g.tag)).toEqual(['local', 'prod', 'alpha', 'zeta'])
  })
})

describe('matchesTag', () => {
  it('keeps everything for an empty filter', () => {
    expect(matchesTag({}, '')).toBe(true)
    expect(matchesTag({ tag: 'prod' }, '')).toBe(true)
  })

  it('matches ignoring case and spaces around the tag', () => {
    expect(matchesTag({ tag: 'Prod ' }, 'prod')).toBe(true)
    expect(matchesTag({ tag: 'staging' }, 'prod')).toBe(false)
    expect(matchesTag({}, 'prod')).toBe(false)
  })
})

describe('activeFilter', () => {
  const groups = tagGroups([{ tag: 'prod' }, { tag: 'staging' }])

  it('keeps a chosen tag that is still in use, ignoring case', () => {
    expect(activeFilter(groups, 'Prod')).toBe('Prod')
  })

  it('drops a tag that no server has any more', () => {
    expect(activeFilter(groups, 'qa')).toBe('')
  })

  it('does not filter with fewer than two tags to choose from', () => {
    expect(activeFilter(tagGroups([{ tag: 'prod' }]), 'prod')).toBe('')
    expect(activeFilter([], 'prod')).toBe('')
  })
})
