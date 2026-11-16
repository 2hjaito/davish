import { describe, expect, it } from 'vitest'

import { aroundStart, mockAround, mockLogSearch, PAGE } from './history'

describe('aroundStart', () => {
  it('centres the hit, never before line 1', () => {
    expect(aroundStart(1)).toBe(1)
    expect(aroundStart(30)).toBe(1)
    expect(aroundStart(1000)).toBe(1000 - PAGE / 2)
  })
})

describe('mockLogSearch', () => {
  it('numbers matching lines from 1', () => {
    const r = mockLogSearch(['ok', 'Timeout here', 'fine', 'timeout again'], 'timeout', false)
    expect(r.lines).toEqual([
      { number: 2, text: 'Timeout here' },
      { number: 4, text: 'timeout again' },
    ])
    expect(mockLogSearch(['Timeout'], 'timeout', true).lines).toEqual([])
  })
})

describe('mockAround', () => {
  it('pages and tells the end', () => {
    const lines = Array.from({ length: 150 }, (_, i) => `l${i + 1}`)
    const p = mockAround(lines, 101, 100)
    expect(p.lines[0]).toEqual({ number: 101, text: 'l101' })
    expect(p.lines).toHaveLength(50)
    expect(p.eof).toBe(true)
    expect(mockAround(lines, 1, 100).eof).toBe(false)
  })
})
