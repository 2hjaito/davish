import { describe, expect, it } from 'vitest'

import { PANES, searchPanes } from './panes'

describe('searchPanes', () => {
  it('lists every pane for an empty query', () => {
    expect(searchPanes('  ')).toEqual([...PANES])
  })

  it('finds a pane by the name of one of its rows', () => {
    // "Cỡ chữ" is a Terminal row.
    expect(searchPanes('cỡ chữ')).toEqual(['terminal'])
  })

  it('ignores case and Vietnamese accents', () => {
    expect(searchPanes('BAO MAT')).toContain('security')
    expect(searchPanes('dock')).toContain('dock')
  })

  it('finds nothing for unknown words', () => {
    expect(searchPanes('zzzz')).toEqual([])
  })
})
