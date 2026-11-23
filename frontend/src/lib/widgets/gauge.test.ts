import { describe, expect, it } from 'vitest'

import { cellsFor, dotCells, ringDash, splitUnit } from './gauge'

describe('ringDash', () => {
  const r = 20
  const c = 2 * Math.PI * r
  const filled = (s: string) => Number(s.split(' ')[0])

  it('fills none, half and all of the ring', () => {
    expect(filled(ringDash(0, r))).toBe(0)
    expect(filled(ringDash(0.5, r))).toBeCloseTo(c / 2, 1)
    expect(filled(ringDash(1, r))).toBeCloseTo(c, 1)
  })

  it('keeps the value in range, and treats junk as empty', () => {
    expect(filled(ringDash(-3, r))).toBe(0)
    expect(filled(ringDash(7, r))).toBeCloseTo(c, 1)
    expect(filled(ringDash(Number.NaN, r))).toBe(0)
  })
})

describe('cellsFor', () => {
  it('rounds a share of n cells', () => {
    expect(cellsFor(0.5, 10)).toBe(5)
    expect(cellsFor(0.26, 10)).toBe(3)
  })

  it('shows a small reading, and does not call a near full one full', () => {
    expect(cellsFor(0.001, 50)).toBe(1)
    expect(cellsFor(0.999, 50)).toBe(49)
  })

  it('is none for zero and all for whole', () => {
    expect(cellsFor(0, 50)).toBe(0)
    expect(cellsFor(1, 50)).toBe(50)
    expect(cellsFor(0.5, 0)).toBe(0)
    expect(cellsFor(Number.NaN, 10)).toBe(0)
  })
})

describe('dotCells', () => {
  it('lays the parts in order and fills the rest with empty', () => {
    expect(
      dotCells(6, [
        { kind: 'a', count: 2 },
        { kind: 'b', count: 1 },
      ]),
    ).toEqual(['a', 'a', 'b', 'empty', 'empty', 'empty'])
  })

  it('cuts at n, and is all empty without parts', () => {
    expect(dotCells(2, [{ kind: 'a', count: 5 }])).toEqual(['a', 'a'])
    expect(dotCells(3, [])).toEqual(['empty', 'empty', 'empty'])
    expect(dotCells(0, [{ kind: 'a', count: 1 }])).toEqual([])
  })
})

describe('splitUnit', () => {
  it('splits a size into its number and its unit', () => {
    expect(splitUnit('42,4 GB')).toEqual({ value: '42,4', unit: 'GB' })
    expect(splitUnit('431 GB')).toEqual({ value: '431', unit: 'GB' })
    expect(splitUnit('6 MB')).toEqual({ value: '6', unit: 'MB' })
  })

  it('keeps a text with no unit as the number, and does not fail on an empty one', () => {
    expect(splitUnit('12')).toEqual({ value: '12', unit: '' })
    expect(splitUnit('')).toEqual({ value: '', unit: '' })
  })
})
