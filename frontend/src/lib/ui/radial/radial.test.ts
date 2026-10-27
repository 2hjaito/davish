import { describe, expect, it } from 'vitest'

import { bearing, hitTest, NO_HIT, petalSpan, polar, RADII, sectorPath } from './radial'

const none = () => 0

describe('bearing', () => {
  it('goes clockwise from the top in screen coordinates', () => {
    expect(bearing(0, -1)).toBe(0)
    expect(bearing(1, 0)).toBe(90)
    expect(bearing(0, 1)).toBe(180)
    expect(bearing(-1, 0)).toBe(270)
  })
})

describe('hitTest with four petals', () => {
  const mid = (RADII.inner + RADII.outer) / 2

  it('finds the petal by direction', () => {
    expect(hitTest(0, -mid, 4, none)).toEqual({ petal: 0, sub: null })
    expect(hitTest(mid, 0, 4, none)).toEqual({ petal: 1, sub: null })
    expect(hitTest(0, mid, 4, none)).toEqual({ petal: 2, sub: null })
    expect(hitTest(-mid, 0, 4, none)).toEqual({ petal: 3, sub: null })
    // Just left of the top still belongs to the top petal.
    expect(hitTest(-10, -mid, 4, none).petal).toBe(0)
  })

  it('cancels in the center and outside', () => {
    expect(hitTest(0, -10, 4, none)).toEqual(NO_HIT)
    expect(hitTest(0, -(RADII.outer + 10), 4, none)).toEqual(NO_HIT)
    expect(hitTest(0, -(RADII.ring + 1), 4, () => 3)).toEqual(NO_HIT)
  })

  it('splits the outer ring of a petal between its choices', () => {
    const r = (RADII.outer + RADII.ring) / 2
    const subs = (i: number) => (i === 3 ? 3 : 0)
    // Petal 3 spans bearings 225–315; its three choices 30° each, clockwise.
    const at = (deg: number) => polar(0, 0, r, deg)
    for (const [deg, sub] of [
      [235, 0],
      [270, 1],
      [305, 2],
    ] as const) {
      const p = at(deg)
      expect(hitTest(p.x, p.y, 4, subs)).toEqual({ petal: 3, sub })
    }
    const top = at(0)
    expect(hitTest(top.x, top.y, 4, subs)).toEqual(NO_HIT)
  })
})

describe('petalSpan', () => {
  it('centers each petal on its direction', () => {
    expect(petalSpan(0, 4)).toEqual([-45, 45])
    expect(petalSpan(3, 4)).toEqual([225, 315])
  })
})

describe('sectorPath', () => {
  it('draws a closed ring sector', () => {
    const d = sectorPath(100, 100, 10, 20, 0, 90)
    expect(d).toBe(
      'M 100.00 80.00 A 20 20 0 0 1 120.00 100.00 L 110.00 100.00 A 10 10 0 0 0 100.00 90.00 Z',
    )
  })
})
