// The grid desktop icons snap to, filled from the top left: notifications
// land on the top right, and would cover an icon's stop button there.

export const CELL_W = 96
export const CELL_H = 100
const PAD = 10

export interface Cell {
  col: number
  row: number
}

export interface Area {
  x: number
  y: number
  w: number
  h: number
}

export interface Grid {
  cols: number
  rows: number
}

export function gridOf(area: Area): Grid {
  return {
    cols: Math.max(1, Math.floor((area.w - PAD * 2) / CELL_W)),
    rows: Math.max(1, Math.floor((area.h - PAD * 2) / CELL_H)),
  }
}

/** Top left of a cell. */
export function cellPosition(c: Cell, area: Area): { x: number; y: number } {
  return {
    x: area.x + PAD + CELL_W * c.col,
    y: area.y + PAD + CELL_H * c.row,
  }
}

/** The cell a point falls in, kept inside the grid. */
export function cellAt(x: number, y: number, area: Area, grid: Grid): Cell {
  const col = Math.floor((x - area.x - PAD) / CELL_W)
  const row = Math.floor((y - area.y - PAD) / CELL_H)
  return {
    col: Math.min(Math.max(col, 0), grid.cols - 1),
    row: Math.min(Math.max(row, 0), grid.rows - 1),
  }
}

const key = (c: Cell) => `${c.col}:${c.row}`

/**
 * The free cell nearest want, the next one down the column on a tie. A
 * full grid gives want back: icons overlap rather than disappear.
 */
export function freeCell(want: Cell, taken: readonly Cell[], grid: Grid): Cell {
  const start = {
    col: Math.min(want.col, grid.cols - 1),
    row: Math.min(want.row, grid.rows - 1),
  }
  const used = new Set(taken.map(key))
  let best: Cell | null = null
  let bestDistance = Infinity
  for (let col = 0; col < grid.cols; col++) {
    for (let row = 0; row < grid.rows; row++) {
      if (used.has(key({ col, row }))) continue
      const d = Math.hypot(col - start.col, row - start.row)
      if (d < bestDistance) {
        best = { col, row }
        bestDistance = d
      }
    }
  }
  return best ?? start
}

/**
 * Where icons show in a grid that may be smaller than when they were
 * placed: each keeps its cell if it fits, the rest move to a free one.
 * The stored cells stay, so widening the window brings them back.
 */
export function layout<T extends Cell>(items: readonly T[], grid: Grid): Cell[] {
  const inside = (c: Cell) => c.col < grid.cols && c.row < grid.rows
  const placed: Cell[] = items.filter(inside)
  return items.map((item) => {
    if (inside(item)) return item
    const cell = freeCell(item, placed, grid)
    placed.push(cell)
    return cell
  })
}
