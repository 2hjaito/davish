import { describe, expect, it } from 'vitest'

import { approach, magnify, MAX_SCALE, nearestSlot, slotCenter, slotCenters } from './magnify'

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

  it('settles within a given tolerance', () => {
    expect(approach(0.4, 0, 16, 0.5)).toBe(0)
    expect(approach(40, 0, 16, 0.5)).toBeGreaterThan(0.5)
  })
})

describe('slotCenters', () => {
  const tile = 56

  it('matches the resting centers when nothing is magnified', () => {
    const c = slotCenters(5, [1, 1, 1, 1, 1], step, tile)
    c.forEach((x, i) => expect(x).toBeCloseTo(slotCenter(i, 5, step)))
  })

  it('treats missing scales as 1', () => {
    expect(slotCenters(3, [], step, tile)).toEqual([-step, 0, step])
  })

  it('pushes neighbours away from a magnified tile and keeps the dock centered', () => {
    // The middle tile grows by 0.2 × 56 = 11.2px: half on each side.
    const c = slotCenters(3, [1, 1.2, 1], step, tile)
    expect(c[0]).toBeCloseTo(-step - 5.6)
    expect(c[1]).toBeCloseTo(0)
    expect(c[2]).toBeCloseTo(step + 5.6)
  })

  it('shifts every tile when the growth is off center', () => {
    // The first tile grows by 11.2px; the dock grows 5.6px each way.
    const c = slotCenters(2, [1.2, 1], step, tile)
    expect(c[0]).toBeCloseTo(-step / 2)
    expect(c[1]).toBeCloseTo(step / 2 + 5.6)
  })
})

describe('nearestSlot', () => {
  const c = [-100, -50, 0, 50, 100]

  it('finds the slot whose center is nearest', () => {
    expect(nearestSlot(c, -1, 0, 4)).toBe(2)
    expect(nearestSlot(c, 24, 0, 4)).toBe(2)
    expect(nearestSlot(c, 26, 0, 4)).toBe(3)
    expect(nearestSlot(c, -76, 0, 4)).toBe(0)
  })

  it('stays within min and max', () => {
    expect(nearestSlot(c, -500, 1, 3)).toBe(1)
    expect(nearestSlot(c, 500, 1, 3)).toBe(3)
  })
})
