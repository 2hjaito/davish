import { describe, expect, it } from 'vitest'

import { approach, magnify, MAX_SCALE } from './magnify'

const step = 66

describe('magnify', () => {
  it('peaks under the pointer and is symmetric around it', () => {
    const s = magnify(0, 5, step)
    expect(s[2]).toBeCloseTo(MAX_SCALE)
    expect(s[1]).toBeCloseTo(s[3] ?? 0)
    expect(s[1]).toBeGreaterThan(1)
    expect(s[1]).toBeLessThan(MAX_SCALE)
  })

  it('leaves tiles out of reach untouched', () => {
    expect(magnify(0, 7, step)).toEqual(expect.arrayContaining([1]))
    expect(magnify(0, 7, step)[0]).toBe(1)
  })

  it('changes continuously as the pointer moves between tiles', () => {
    const a = magnify(-0.5 * step, 4, step)
    const b = magnify(-0.5 * step + 1, 4, step)
    a.forEach((v, i) => expect(Math.abs(v - (b[i] ?? 0))).toBeLessThan(0.01))
  })
})

describe('approach', () => {
  it('moves part of the way and settles exactly on the target', () => {
    const next = approach(1, 1.2, 16)
    expect(next).toBeGreaterThan(1)
    expect(next).toBeLessThan(1.2)
    expect(approach(1.19999, 1.2, 16)).toBe(1.2)
  })
})
