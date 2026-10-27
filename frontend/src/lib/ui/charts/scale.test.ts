import { describe, expect, it } from 'vitest'

import { areaPath, linePath, nearestIndex, niceMax } from './scale'

describe('niceMax', () => {
  it.each([
    [0, {}, 1],
    [0.3, {}, 1],
    [7, {}, 10],
    [23, {}, 50],
    [100, {}, 100],
    [101, {}, 200],
    [0, { floor: 100 }, 100],
    [1500, { binary: true }, 2 * 1024],
    [700 * 1024, { binary: true }, 1000 * 1024],
    [1.3 * 1024 ** 2, { binary: true }, 2 * 1024 ** 2],
    [9 * 1024 ** 3, { binary: true }, 10 * 1024 ** 3],
    [10, { binary: true, floor: 1024 }, 1024],
  ])('niceMax(%d, %o) = %d', (v, opts, want) => {
    expect(niceMax(v, opts)).toBe(want)
  })
})

describe('paths', () => {
  const points = [
    { x: 0, y: 10 },
    { x: 1, y: 5 },
    { x: 2, y: 8 },
    { x: 3, y: 2 },
  ]

  it('breaks the line where samples are missing', () => {
    expect(linePath(points, [0, 2, 20, 22], 5)).toBe('M0.0 10.0L1.0 5.0M2.0 8.0L3.0 2.0')
  })

  it('closes each area segment down to the baseline', () => {
    expect(areaPath(points, [0, 2, 20, 22], 5, 12)).toBe(
      'M0.0 10.0L1.0 5.0L1.0 12L0.0 12Z' + 'M2.0 8.0L3.0 2.0L3.0 12L2.0 12Z',
    )
  })
})

describe('nearestIndex', () => {
  const times = [0, 10, 20, 30]
  it.each([
    [-5, 0],
    [4, 0],
    [6, 1],
    [24, 2],
    [25, 3], // a tie goes to the newer sample
    [26, 3],
    [99, 3],
  ])('nearest to %d is index %d', (t, want) => {
    expect(nearestIndex(times, t)).toBe(want)
  })

  it('is -1 without data', () => {
    expect(nearestIndex([], 5)).toBe(-1)
  })
})
