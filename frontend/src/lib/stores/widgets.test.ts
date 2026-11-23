import { beforeEach, describe, expect, it, vi } from 'vitest'

import { cellsOf, footprint, MAX_WIDGETS } from '../widgets/layout'
import { widgets } from './widgets.svelte'

const S = 'server-1'
const T = 'server-2'
const grid = { cols: 8, rows: 6 }

const memory = new Map<string, string>()
vi.stubGlobal('localStorage', {
  getItem: (key: string) => memory.get(key) ?? null,
  setItem: (key: string, value: string) => memory.set(key, value),
})

const saved = () => JSON.parse(memory.get('davish:widgets') ?? '{}') as Record<string, unknown[]>
const cells = (id: string, server = S) => {
  const item = widgets.of(server).find((w) => w.id === id)!
  return cellsOf(footprint(item, grid)).map((c) => `${c.col}:${c.row}`)
}

beforeEach(() => {
  for (const id of [S, T]) widgets.of(id).forEach((w) => widgets.remove(id, w.id))
  memory.clear()
})

describe('add', () => {
  it('puts the first widget at the top right and the next below it', () => {
    const a = widgets.add(S, 'monitor', 'S', grid)!
    const b = widgets.add(S, 'monitor', 'S', grid)!
    expect(a).toMatchObject({ col: 0, row: 0 })
    expect(b).toMatchObject({ col: 0, row: 1 })
    expect(a.id).not.toBe(b.id)
  })

  it('never overlaps, and saves what it adds', () => {
    for (let i = 0; i < 6; i++) widgets.add(S, 'monitor', i % 2 ? 'M' : 'S', grid)
    const all = widgets.of(S).flatMap((w) => cells(w.id))
    expect(new Set(all).size).toBe(all.length)
    expect(saved()[S]).toHaveLength(widgets.of(S).length)
  })

  it('keeps each server to its own desktop', () => {
    widgets.add(S, 'monitor', 'S', grid)
    expect(widgets.of(T)).toEqual([])
    widgets.add(T, 'monitor', 'M', grid)
    expect(widgets.of(S)).toHaveLength(1)
    expect(widgets.of(T)).toHaveLength(1)
  })

  it('gives null when the desktop has no room, or is at the limit', () => {
    expect(widgets.add(S, 'monitor', 'M', { cols: 1, rows: 6 })).toBeNull()
    for (let i = 0; i < MAX_WIDGETS; i++) widgets.add(S, 'monitor', 'S', { cols: 20, rows: 20 })
    expect(widgets.of(S)).toHaveLength(MAX_WIDGETS)
    expect(widgets.add(S, 'monitor', 'S', { cols: 20, rows: 20 })).toBeNull()
  })
})

describe('move', () => {
  it('moves to the free place nearest where it was dropped', () => {
    const a = widgets.add(S, 'monitor', 'S', grid)!
    widgets.move(S, a.id, { col: 3, row: 2 }, grid)
    expect(widgets.of(S)[0]).toMatchObject({ col: 3, row: 2 })
  })

  it('does not land on another widget', () => {
    const a = widgets.add(S, 'monitor', 'S', grid)!
    const b = widgets.add(S, 'monitor', 'S', grid)!
    widgets.move(S, b.id, { col: 0, row: 0 }, grid) // a is there
    expect(cells(b.id).some((c) => cells(a.id).includes(c))).toBe(false)
  })

  it('keeps the order widgets were added in', () => {
    const a = widgets.add(S, 'monitor', 'S', grid)!
    const b = widgets.add(S, 'monitor', 'S', grid)!
    widgets.move(S, a.id, { col: 4, row: 3 }, grid)
    expect(widgets.of(S).map((w) => w.id)).toEqual([a.id, b.id])
  })

  it('ignores an id it does not know', () => {
    widgets.move(S, 'nope', { col: 1, row: 1 }, grid)
    expect(widgets.of(S)).toEqual([])
  })
})

describe('resize and remove', () => {
  it('resizes in place, or to where it fits', () => {
    const a = widgets.add(S, 'monitor', 'S', grid)!
    widgets.resize(S, a.id, 'M', grid)
    expect(widgets.of(S)[0]).toMatchObject({ id: a.id, size: 'M', col: 0, row: 0 })
  })

  it('does not grow onto a neighbour', () => {
    const a = widgets.add(S, 'monitor', 'S', grid)!
    const b = widgets.add(S, 'monitor', 'S', grid)!
    widgets.resize(S, a.id, 'M', grid)
    expect(cells(a.id).some((c) => cells(b.id).includes(c))).toBe(false)
  })

  it('removes a widget and forgets a server with none left', () => {
    const a = widgets.add(S, 'monitor', 'S', grid)!
    widgets.remove(S, a.id)
    expect(widgets.of(S)).toEqual([])
    expect(saved()[S]).toBeUndefined()
  })
})

describe('reserved', () => {
  it('lists the cells under the widgets of that server only', () => {
    widgets.add(S, 'monitor', 'M', grid)
    expect(widgets.reserved(S, grid)).toHaveLength(4)
    expect(widgets.reserved(T, grid)).toEqual([])
  })
})
