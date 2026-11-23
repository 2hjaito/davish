import { describe, expect, it } from 'vitest'

import { type Grid, gridOf } from '../shell/iconGrid'
import {
  cellsOf,
  countKinds,
  footprint,
  footprintBox,
  footprintAt,
  layoutWidgets,
  MAX_WIDGETS,
  placeWidget,
  reservedCells,
  sanitizeWidgets,
  TOP_ROWS,
  wantFor,
  type WidgetItem,
} from './layout'

const grid: Grid = { cols: 10, rows: 6 }
const w = (id: string, col: number, row: number, size: 'S' | 'M' = 'S'): WidgetItem => ({
  id,
  kind: 'monitor',
  size,
  col,
  row,
})

describe('footprint', () => {
  it('counts columns from the right edge and rows from below the top rows', () => {
    // S is 2 cells wide: col 0 is the two rightmost columns, 8 and 9.
    expect(footprint(w('a', 0, 0), grid)).toEqual({ col: 8, row: TOP_ROWS, cols: 2, rows: 1 })
    expect(footprint(w('a', 1, 2, 'M'), grid)).toEqual({
      col: 7,
      row: 2 + TOP_ROWS,
      cols: 2,
      rows: 2,
    })
  })

  it('lists every cell it covers', () => {
    expect(cellsOf(footprintAt(0, 0, 'M', grid))).toEqual([
      { col: 8, row: TOP_ROWS },
      { col: 8, row: TOP_ROWS + 1 },
      { col: 9, row: TOP_ROWS },
      { col: 9, row: TOP_ROWS + 1 },
    ])
  })
})

describe('placeWidget', () => {
  it('takes the place nearest the top right when nothing is taken', () => {
    expect(placeWidget({ col: 0, row: 0 }, 'S', new Set(), grid)).toEqual({ col: 0, row: 0 })
  })

  it('skips cells that are taken, the next one down the column on a tie', () => {
    const taken = new Set(cellsOf(footprintAt(0, 0, 'S', grid)).map((c) => `${c.col}:${c.row}`))
    expect(placeWidget({ col: 0, row: 0 }, 'S', taken, grid)).toEqual({ col: 0, row: 1 })
  })

  it('needs the whole footprint free, so M does not fit beside a taken cell', () => {
    // The cell under the top rows at the far right is taken: M cannot start in row 0.
    const taken = new Set(['9:1'])
    const spot = placeWidget({ col: 0, row: 0 }, 'M', taken, grid)
    expect(spot).not.toBeNull()
    const cells = cellsOf(footprintAt(spot!.col, spot!.row, 'M', grid))
    expect(cells.some((c) => c.col === 9 && c.row === 1)).toBe(false)
  })

  it('gives null when there is no room', () => {
    expect(placeWidget({ col: 0, row: 0 }, 'M', new Set(), { cols: 1, rows: 6 })).toBeNull()
    expect(placeWidget({ col: 0, row: 0 }, 'M', new Set(), { cols: 10, rows: 1 })).toBeNull()
  })
})

describe('layoutWidgets', () => {
  it('keeps each widget where it was put when all fit', () => {
    const items = [w('a', 0, 0), w('b', 0, 1), w('c', 2, 0, 'M')]
    const placed = layoutWidgets(items, grid)
    expect(placed.map((p) => p.at)).toEqual(items.map((i) => footprint(i, grid)))
  })

  it('moves a widget a narrower window cannot hold to a free place, and hides one with none', () => {
    // A 3-column grid holds one S widget per row; the third in a row is moved down.
    const narrow: Grid = { cols: 3, rows: 4 }
    const items = [w('a', 0, 0), w('b', 5, 0), w('c', 0, 0)]
    const placed = layoutWidgets(items, narrow)
    expect(placed[0]?.at).toEqual(footprint(items[0]!, narrow))
    // b (col 5 from the right) is outside a 3-column grid, c overlaps a: both are moved or hidden
    const spots = placed.map((p) => p.at)
    const seen = new Set<string>()
    for (const f of spots) {
      if (!f) continue
      for (const c of cellsOf(f)) {
        const k = `${c.col}:${c.row}`
        expect(seen.has(k)).toBe(false)
        seen.add(k)
      }
    }
  })

  it('hides a widget when the grid is full', () => {
    const tiny: Grid = { cols: 2, rows: 1 } // one S widget fits
    const placed = layoutWidgets([w('a', 0, 0), w('b', 0, 0)], tiny)
    expect(placed[0]?.at).not.toBeNull()
    expect(placed[1]?.at).toBeNull()
  })

  it('does not let a moved widget take the place of one that did not have to move', () => {
    const items = [w('a', 0, 0), w('b', 0, 0), w('c', 0, 1)] // b overlaps a; c is fine where it is
    const placed = layoutWidgets(items, grid)
    expect(placed[2]?.at).toEqual(footprint(items[2]!, grid))
  })

  it('never changes what is stored', () => {
    const items = [w('a', 9, 9)]
    const copy = structuredClone(items)
    layoutWidgets(items, grid)
    expect(items).toEqual(copy)
  })
})

describe('reservedCells', () => {
  it('lists the cells of the widgets that show, none for a hidden one', () => {
    expect(reservedCells([w('a', 0, 0)], grid)).toHaveLength(2)
    expect(reservedCells([w('a', 0, 0), w('b', 0, 0)], { cols: 2, rows: 1 })).toHaveLength(2)
    expect(reservedCells([], grid)).toEqual([])
  })
})

describe('sanitizeWidgets', () => {
  const ok = { id: 'a1', kind: 'monitor', size: 'S', col: 0, row: 1 }

  it('keeps a valid file as it is', () => {
    expect(sanitizeWidgets({ srv: [ok] })).toEqual({ srv: [ok] })
  })

  it.each([null, undefined, 'x', 5, [], [ok]])('gives nothing for %j', (junk) => {
    expect(sanitizeWidgets(junk)).toEqual({})
  })

  it('drops a widget of a kind or size it does not know, or with bad numbers', () => {
    const bad = [
      { ...ok, id: 'b', kind: 'cryptominer' },
      { ...ok, id: 'c', size: 'XL' },
      { ...ok, id: 'd', col: -1 },
      { ...ok, id: 'e', col: 1.5 },
      { ...ok, id: 'f', row: 'x' },
      { ...ok, id: 'g', col: 123 },
      { ...ok, id: '' },
      { ...ok, id: 'x'.repeat(41) },
      null,
      'str',
    ]
    expect(sanitizeWidgets({ srv: [ok, ...bad] })).toEqual({ srv: [ok] })
  })

  it('keeps the first of repeated ids and cuts a desktop at MAX_WIDGETS', () => {
    const many = Array.from({ length: MAX_WIDGETS + 5 }, (_, i) => ({ ...ok, id: `w${i}` }))
    expect(sanitizeWidgets({ srv: many }).srv).toHaveLength(MAX_WIDGETS)
    expect(sanitizeWidgets({ srv: [ok, { ...ok, col: 3 }] }).srv).toEqual([ok])
  })

  it('leaves out a server whose widgets are all invalid', () => {
    expect(sanitizeWidgets({ srv: [{ ...ok, kind: 'x' }], other: [ok] })).toEqual({ other: [ok] })
    expect(sanitizeWidgets({ srv: 'nope' })).toEqual({})
  })
})

describe('wantFor', () => {
  it('is the inverse of footprint for the top left cell of a widget', () => {
    for (const item of [w('a', 0, 0), w('b', 3, 2, 'M'), w('c', 1, 1)]) {
      const f = footprint(item, grid)
      expect(wantFor({ col: f.col, row: f.row }, item.size, grid)).toEqual({
        col: item.col,
        row: item.row,
      })
    }
  })

  it('never goes negative, whatever the drop', () => {
    expect(wantFor({ col: 9, row: 0 }, 'S', grid)).toEqual({ col: 0, row: 0 })
    expect(wantFor({ col: 0, row: 0 }, 'S', grid).row).toBe(0)
  })
})

describe('countKinds', () => {
  it('counts each kind, zero for one that is not there', () => {
    expect(countKinds([])).toEqual({ monitor: 0, docker: 0, services: 0, disk: 0 })
    expect(countKinds([w('a', 0, 0), w('b', 0, 1), w('c', 2, 0, 'M')])).toEqual({
      monitor: 3,
      docker: 0,
      services: 0,
      disk: 0,
    })
  })
})

describe('footprintBox margins', () => {
  // The top right widget keeps the same distance from the right edge and from the menu bar,
  // however wide the window is: the grid's leftover width does not show on the right.
  it.each([1000, 1033, 1440, 1517])(
    'is as far from the right edge as from the top at %ipx',
    (width) => {
      const area = { x: 0, y: 30, w: width, h: 700 }
      const g = gridOf(area)
      const box = footprintBox(footprintAt(0, 0, 'S', g), area)
      expect(area.w - (box.x + box.w)).toBe(box.y - area.y)
    },
  )
})
