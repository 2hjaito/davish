import { describe, expect, it } from 'vitest'

import { PANES, searchPanes } from './panes'

describe('searchPanes', () => {
  it('lists every pane for an empty query', () => {
    expect(searchPanes('  ')).toEqual([...PANES])
  })

  it('finds a pane by the name of one of its rows', () => {
    // "Prompt Davish" is a Terminal row; "Cỡ chữ" is one of Terminal and of Editor.
    expect(searchPanes('prompt')).toEqual(['terminal'])
    expect(searchPanes('cỡ chữ')).toEqual(['terminal', 'editor'])
  })

  it('finds the terminal rows added for solid background and scrollback', () => {
    expect(searchPanes('cuộn lại')).toEqual(['terminal'])
  })

  it('has a Files pane, found by its row', () => {
    expect(searchPanes('file ẩn')).toEqual(['files'])
  })

  it('ignores case and Vietnamese accents', () => {
    expect(searchPanes('BAO MAT')).toContain('security')
    expect(searchPanes('dock')).toContain('dock')
  })

  it('puts the account first and Davish last, with the lock under the account', () => {
    expect(PANES[0]).toBe('account')
    expect(PANES[PANES.length - 1]).toBe('about')
    expect(searchPanes('khóa app')).toContain('account')
    expect(searchPanes('kiểm tra ngay')).toEqual(['about'])
  })

  it('finds nothing for unknown words', () => {
    expect(searchPanes('zzzz')).toEqual([])
  })
})
