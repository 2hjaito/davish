import { describe, expect, it } from 'vitest'
import { CATEGORIES, CATEGORY, gridMove, grouped } from './categories'
import { apps } from './registry'

describe('CATEGORY', () => {
  it('puts every app in a known group', () => {
    for (const a of apps) expect(CATEGORIES).toContain(CATEGORY[a.id])
  })
})

describe('grouped', () => {
  it('keeps group and app order and drops empty groups', () => {
    const list = [{ id: 'git' }, { id: 'terminal' }, { id: 'files' }] as const
    expect(grouped(list)).toEqual([
      { category: 'files', apps: [{ id: 'terminal' }, { id: 'files' }] },
      { category: 'dev', apps: [{ id: 'git' }] },
    ])
  })
})

describe('gridMove', () => {
  // Two grids of 5 columns: 0-6 (rows 0-4, 5-6), then 7-9.
  const sizes = [7, 3]
  it('steps left and right across grids, stopping at the ends', () => {
    expect(gridMove(sizes, 6, 'ArrowRight', 5)).toBe(7)
    expect(gridMove(sizes, 9, 'ArrowRight', 5)).toBe(9)
    expect(gridMove(sizes, 0, 'ArrowLeft', 5)).toBe(0)
  })
  it('keeps the column moving down, onto the last tile of a short row', () => {
    expect(gridMove(sizes, 1, 'ArrowDown', 5)).toBe(6)
    expect(gridMove(sizes, 4, 'ArrowDown', 5)).toBe(6)
    expect(gridMove(sizes, 5, 'ArrowDown', 5)).toBe(7)
    expect(gridMove(sizes, 8, 'ArrowDown', 5)).toBe(8)
  })
  it('moves up into the previous grid', () => {
    expect(gridMove(sizes, 9, 'ArrowUp', 5)).toBe(6)
    expect(gridMove(sizes, 7, 'ArrowUp', 5)).toBe(5)
    expect(gridMove(sizes, 6, 'ArrowUp', 5)).toBe(1)
    expect(gridMove(sizes, 2, 'ArrowUp', 5)).toBe(2)
  })
})
