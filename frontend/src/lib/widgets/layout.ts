// Where widgets sit on a server's desktop (ADR 0026). They snap to the same
// cells as the desktop icons but count from the right edge, so the icons fill
// from the top left and the widgets from the top right, and neither moves the
// other until the window is too narrow for both. All pure.
import { type Area, type Cell, cellPosition, CELL_H, CELL_W, type Grid } from '../shell/iconGrid'
import { isWidgetKind, WIDGET_KINDS, type WidgetKind } from './kinds'

export type WidgetSize = 'S' | 'M'

export const WIDGET_SIZES: Record<WidgetSize, { cols: number; rows: number }> = {
  S: { cols: 2, rows: 1 },
  M: { cols: 2, rows: 2 },
}

/** Rows kept clear at the top: notifications land there (see iconGrid.ts). */
export const TOP_ROWS = 1

/** More than this on one desktop is a clutter, and a file with more is cut. */
export const MAX_WIDGETS = 12

/** A widget on a server's desktop. col counts from the right edge, row from below TOP_ROWS. */
export interface WidgetItem extends Cell {
  id: string
  kind: WidgetKind
  size: WidgetSize
}

/** The cells a widget covers, in the grid's own columns (from the left). */
export interface Footprint {
  col: number
  row: number
  cols: number
  rows: number
}

/** The cells of a widget of this size at col from the right and row below the top rows. */
export function footprintAt(col: number, row: number, size: WidgetSize, grid: Grid): Footprint {
  const { cols, rows } = WIDGET_SIZES[size]
  return { col: grid.cols - col - cols, row: row + TOP_ROWS, cols, rows }
}

export function footprint(item: WidgetItem, grid: Grid): Footprint {
  return footprintAt(item.col, item.row, item.size, grid)
}

function fits(f: Footprint, grid: Grid): boolean {
  return f.col >= 0 && f.row >= 0 && f.col + f.cols <= grid.cols && f.row + f.rows <= grid.rows
}

const key = (c: Cell) => `${c.col}:${c.row}`

export function cellsOf(f: Footprint): Cell[] {
  const out: Cell[] = []
  for (let col = f.col; col < f.col + f.cols; col++)
    for (let row = f.row; row < f.row + f.rows; row++) out.push({ col, row })
  return out
}

/**
 * The position (col from the right, row below the top rows) nearest want
 * where a widget of this size fits among the taken cells; null when nowhere.
 */
export function placeWidget(
  want: Cell,
  size: WidgetSize,
  taken: ReadonlySet<string>,
  grid: Grid,
): Cell | null {
  const { cols, rows } = WIDGET_SIZES[size]
  const maxCol = grid.cols - cols
  const maxRow = grid.rows - TOP_ROWS - rows
  let best: Cell | null = null
  let bestDistance = Infinity
  for (let col = 0; col <= maxCol; col++) {
    for (let row = 0; row <= maxRow; row++) {
      const f = footprintAt(col, row, size, grid)
      if (cellsOf(f).some((c) => taken.has(key(c)))) continue
      const d = Math.hypot(col - want.col, row - want.row)
      if (d < bestDistance) {
        best = { col, row }
        bestDistance = d
      }
    }
  }
  return best
}

export interface Placed {
  item: WidgetItem
  /** Where it shows; null when the window has no room for it (it is kept, hidden). */
  at: Footprint | null
}

/**
 * Where each widget shows in a grid that may be smaller than when they were
 * placed: one that still fits and does not overlap an earlier one stays; the
 * rest move to the nearest free place, or are hidden when there is none. The
 * stored cells never change here, so a wider window brings them back.
 */
export function layoutWidgets(items: readonly WidgetItem[], grid: Grid): Placed[] {
  const taken = new Set<string>()
  const claim = (f: Footprint) => cellsOf(f).forEach((c) => taken.add(key(c)))
  const free = (f: Footprint) => fits(f, grid) && !cellsOf(f).some((c) => taken.has(key(c)))
  // The ones that fit as stored are settled first, in order, so a moved one
  // never takes the place of one that did not have to move.
  const at = new Map<string, Footprint>()
  for (const item of items) {
    const f = footprint(item, grid)
    if (!free(f)) continue
    claim(f)
    at.set(item.id, f)
  }
  return items.map((item) => {
    let f = at.get(item.id) ?? null
    if (!f) {
      const spot = placeWidget(item, item.size, taken, grid)
      if (spot) {
        f = footprintAt(spot.col, spot.row, item.size, grid)
        claim(f)
      }
    }
    return { item, at: f }
  })
}

/** The grid cells the visible widgets cover: what the desktop icons must keep clear of. */
export function reservedCells(items: readonly WidgetItem[], grid: Grid): Cell[] {
  return layoutWidgets(items, grid).flatMap((p) => (p.at ? cellsOf(p.at) : []))
}

/**
 * The position (col from the right, row below the top rows) of a widget of
 * this size whose top left corner sits on grid cell: what a drop means.
 */
export function wantFor(cell: Cell, size: WidgetSize, grid: Grid): Cell {
  const { cols } = WIDGET_SIZES[size]
  return { col: Math.max(0, grid.cols - cell.col - cols), row: Math.max(0, cell.row - TOP_ROWS) }
}

/** Pixel box of a footprint, with a small gap so neighbours do not touch. */
export function footprintBox(
  f: Footprint,
  area: Area,
  gap = 8,
): { x: number; y: number; w: number; h: number } {
  const p = cellPosition(f, area)
  return { x: p.x + gap / 2, y: p.y + gap / 2, w: f.cols * CELL_W - gap, h: f.rows * CELL_H - gap }
}

/** How many widgets of each kind are on a desktop. */
export function countKinds(items: readonly WidgetItem[]): Record<WidgetKind, number> {
  const counts = Object.fromEntries(WIDGET_KINDS.map((k) => [k, 0])) as Record<WidgetKind, number>
  for (const item of items) counts[item.kind]++
  return counts
}

const INT = /^\d{1,2}$/

/**
 * What a settings.json may hold under "widgets", cut down to what is valid: a
 * map of server id to widgets. Unknown kinds, sizes and malformed items are
 * dropped (a newer or damaged file must not break the desktop), ids repeated
 * on one desktop keep the first, and at most MAX_WIDGETS stay per server.
 */
export function sanitizeWidgets(raw: unknown): Record<string, WidgetItem[]> {
  const out: Record<string, WidgetItem[]> = {}
  if (typeof raw !== 'object' || raw === null || Array.isArray(raw)) return out
  for (const [serverID, list] of Object.entries(raw)) {
    if (!Array.isArray(list)) continue
    const ids = new Set<string>()
    const kept: WidgetItem[] = []
    for (const v of list as unknown[]) {
      if (typeof v !== 'object' || v === null) continue
      const { id, kind, size, col, row } = v as Record<string, unknown>
      if (typeof id !== 'string' || id === '' || id.length > 40 || ids.has(id)) continue
      if (!isWidgetKind(kind) || (size !== 'S' && size !== 'M')) continue
      if (!INT.test(String(col)) || !INT.test(String(row))) continue
      ids.add(id)
      kept.push({ id, kind, size, col: Number(col), row: Number(row) })
      if (kept.length === MAX_WIDGETS) break
    }
    if (kept.length > 0) out[serverID] = kept
  }
  return out
}
