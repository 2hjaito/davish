import { newId } from '../../apps/snippets/snippets'
import type { Cell, Grid } from '../shell/iconGrid'
import type { WidgetKind } from '../widgets/kinds'
import {
  cellsOf,
  layoutWidgets,
  MAX_WIDGETS,
  placeWidget,
  reservedCells,
  sanitizeWidgets,
  type WidgetItem,
  type WidgetSize,
} from '../widgets/layout'
import { load, save } from './persist'

const KEY = 'widgets'

const cellKey = (c: Cell) => `${c.col}:${c.row}`

/**
 * The widgets on each server's desktop (ADR 0026): a layout and nothing the
 * server said. Per server, as the desktop icons are, so staging's widgets
 * never show on production.
 */
class Widgets {
  byServer = $state<Record<string, WidgetItem[]>>(sanitizeWidgets(load(KEY, {})))

  of(serverID: string): WidgetItem[] {
    return this.byServer[serverID] ?? []
  }

  /** The cells the visible widgets cover, for the icons to keep clear of. */
  reserved(serverID: string, grid: Grid): Cell[] {
    return reservedCells(this.of(serverID), grid)
  }

  /** Adds a widget at the free place nearest the top right; null when full or no room. */
  add(serverID: string, kind: WidgetKind, size: WidgetSize, grid: Grid): WidgetItem | null {
    const items = this.of(serverID)
    if (items.length >= MAX_WIDGETS) return null
    const taken = new Set(this.#taken(serverID, grid))
    const spot = placeWidget({ col: 0, row: 0 }, size, taken, grid)
    if (!spot) return null
    const made: WidgetItem = { id: newId(), kind, size, ...spot }
    this.#set(serverID, [...items, made])
    return made
  }

  /** Moves a widget to the free place nearest want (col from the right); it stays when none is free. */
  move(serverID: string, id: string, want: Cell, grid: Grid) {
    const item = this.of(serverID).find((w) => w.id === id)
    if (!item) return
    const others = this.of(serverID).filter((w) => w.id !== id)
    const spot = placeWidget(want, item.size, new Set(this.#taken(serverID, grid, id)), grid)
    if (spot)
      this.#set(serverID, [...others, { ...item, ...spot }].sort(byOrder(this.of(serverID))))
  }

  /** Changes a widget's size in place, or to the nearest place it fits. */
  resize(serverID: string, id: string, size: WidgetSize, grid: Grid) {
    const item = this.of(serverID).find((w) => w.id === id)
    if (!item || item.size === size) return
    const spot = placeWidget(item, size, new Set(this.#taken(serverID, grid, id)), grid)
    if (!spot) return
    this.#set(
      serverID,
      this.of(serverID).map((w) => (w.id === id ? { ...w, size, ...spot } : w)),
    )
  }

  remove(serverID: string, id: string) {
    this.#set(
      serverID,
      this.of(serverID).filter((w) => w.id !== id),
    )
  }

  /** Cells the other widgets of serverID take on screen, without the one named. */
  #taken(serverID: string, grid: Grid, except?: string): string[] {
    const others = this.of(serverID).filter((w) => w.id !== except)
    return layoutWidgets(others, grid).flatMap((p) => (p.at ? cellsOf(p.at).map(cellKey) : []))
  }

  #set(serverID: string, items: WidgetItem[]) {
    const next = { ...this.byServer }
    if (items.length > 0) next[serverID] = items
    else delete next[serverID]
    this.byServer = next
    save(KEY, this.byServer)
  }
}

/** Keeps the order widgets were added in when one is moved. */
function byOrder(order: readonly WidgetItem[]) {
  const rank = new Map(order.map((w, i) => [w.id, i]))
  return (a: WidgetItem, b: WidgetItem) => (rank.get(a.id) ?? 0) - (rank.get(b.id) ?? 0)
}

export const widgets = new Widgets()
