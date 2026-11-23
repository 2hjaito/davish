import { describe, expect, it } from 'vitest'

import { cellAt, cellPosition, freeCell, gridOf, layout } from './iconGrid'

const area = { x: 0, y: 30, w: 1000, h: 620 }

describe('gridOf', () => {
  it('fits whole cells and never less than one', () => {
    expect(gridOf(area)).toEqual({ cols: 10, rows: 6 })
    expect(gridOf({ x: 0, y: 0, w: 10, h: 10 })).toEqual({ cols: 1, rows: 1 })
  })
})

describe('cellAt', () => {
  const grid = gridOf(area)

  it('is the inverse of cellPosition', () => {
    for (const c of [
      { col: 0, row: 0 },
      { col: 3, row: 2 },
    ]) {
      const p = cellPosition(c, area)
      expect(cellAt(p.x + 5, p.y + 5, area, grid)).toEqual(c)
    }
  })

  it('stays inside the grid', () => {
    expect(cellAt(-50, -50, area, grid)).toEqual({ col: 0, row: 0 })
    expect(cellAt(5000, 5000, area, grid)).toEqual({ col: 9, row: 5 })
  })
})

describe('freeCell', () => {
  const grid = { cols: 2, rows: 2 }

  it('gives the wanted cell when free', () => {
    expect(freeCell({ col: 1, row: 1 }, [], grid)).toEqual({ col: 1, row: 1 })
  })

  it('moves to the nearest free cell', () => {
    expect(freeCell({ col: 0, row: 0 }, [{ col: 0, row: 0 }], grid)).toEqual({ col: 0, row: 1 })
  })

  it('overlaps rather than loses an icon on a full grid', () => {
    const all = [
      { col: 0, row: 0 },
      { col: 0, row: 1 },
      { col: 1, row: 0 },
      { col: 1, row: 1 },
    ]
    expect(freeCell({ col: 1, row: 0 }, all, grid)).toEqual({ col: 1, row: 0 })
  })
})

describe('layout', () => {
  it('leaves icons that fit where they are', () => {
    const items = [{ col: 0, row: 1 }]
    expect(layout(items, { cols: 3, rows: 3 })).toEqual(items)
  })

  it('brings cut-off icons into free cells without stacking', () => {
    const items = [
      { col: 0, row: 0 },
      { col: 0, row: 5 },
      { col: 0, row: 6 },
    ]
    const cells = layout(items, { cols: 2, rows: 2 })
    expect(new Set(cells.map((c) => `${c.col}:${c.row}`)).size).toBe(3)
    expect(cells[0]).toEqual({ col: 0, row: 0 })
  })
})

describe('layout with reserved cells', () => {
  const grid = { cols: 3, rows: 3 }

  it('moves an icon off a reserved cell and keeps the rest', () => {
    const items = [
      { col: 0, row: 0 },
      { col: 2, row: 0 },
    ]
    const cells = layout(items, grid, [{ col: 2, row: 0 }])
    expect(cells[0]).toEqual({ col: 0, row: 0 })
    expect(cells[1]).not.toEqual({ col: 2, row: 0 })
  })

  it('does not put a moved icon on another reserved cell', () => {
    const reserved = [
      { col: 2, row: 0 },
      { col: 2, row: 1 },
    ]
    const [moved] = layout([{ col: 2, row: 0 }], grid, reserved)
    expect(reserved).not.toContainEqual(moved)
  })

  it('is the old layout when nothing is reserved', () => {
    const items = [{ col: 5, row: 5 }]
    expect(layout(items, grid, [])).toEqual(layout(items, grid))
  })
})
