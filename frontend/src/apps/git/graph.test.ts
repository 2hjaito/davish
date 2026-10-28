import { describe, expect, it } from 'vitest'

import type { Commit } from '../../lib/api/git'
import { layout } from './graph'

const c = (hash: string, ...parents: string[]): Commit => ({
  hash,
  short: hash,
  author: '',
  time: 0,
  subject: hash,
  parents,
  refs: [],
})

describe('layout', () => {
  it('keeps a straight history in one lane', () => {
    const rows = layout([c('c', 'b'), c('b', 'a'), c('a')])
    expect(rows.map((r) => r.lane)).toEqual([0, 0, 0])
    expect(rows[0]?.segments).toEqual([{ from: 0, to: 0, half: 'bottom', color: 0 }])
    expect(rows[1]?.segments).toEqual([
      { from: 0, to: 0, half: 'top', color: 0 },
      { from: 0, to: 0, half: 'bottom', color: 0 },
    ])
    expect(rows[2]?.segments).toEqual([{ from: 0, to: 0, half: 'top', color: 0 }])
  })

  it('opens a lane for a merged branch and closes it at the fork', () => {
    //  m        merge of x into b
    //  |\
    //  | x
    //  b |
    //  |/
    //  a
    const rows = layout([c('m', 'b', 'x'), c('x', 'a'), c('b', 'a'), c('a')])
    expect(rows.map((r) => r.lane)).toEqual([0, 1, 0, 0])
    expect(rows[0]?.segments).toContainEqual({ from: 0, to: 1, half: 'bottom', color: 1 })
    expect(rows[1]?.segments).toContainEqual({ from: 0, to: 0, half: 'full', color: 0 })
    // b and x both wait for a: two lanes join at a.
    expect(rows[3]?.segments).toEqual([
      { from: 0, to: 0, half: 'top', color: 0 },
      { from: 1, to: 0, half: 'top', color: 1 },
    ])
    expect(rows[3]?.width).toBe(2)
  })

  it('gives two branch tips their own lanes', () => {
    const rows = layout([c('f', 'a'), c('g', 'a'), c('a')])
    expect(rows.map((r) => r.lane)).toEqual([0, 1, 0])
  })

  it('reuses a freed lane', () => {
    // x ends at its root, so the next tip y takes its lane again.
    const rows = layout([c('b', 'a'), c('x'), c('a'), c('y', 'z'), c('z')])
    expect(rows[1]?.lane).toBe(1)
    expect(rows[3]?.lane).toBe(0)
  })
})
