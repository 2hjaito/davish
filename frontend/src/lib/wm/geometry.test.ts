import { describe, expect, it } from 'vitest'

import { clampRect, fitRect, MIN_H, MIN_W, resizeRect, snapZone, zoneRect } from './geometry'

const area = { x: 0, y: 30, w: 1000, h: 600 }

describe('resizeRect', () => {
  const r = { x: 100, y: 100, w: 500, h: 400 }

  it('grows from the south-east corner', () => {
    expect(resizeRect(r, 'se', 20, 10)).toEqual({ x: 100, y: 100, w: 520, h: 410 })
  })

  it('keeps the opposite edge fixed when resizing from the north-west', () => {
    expect(resizeRect(r, 'nw', 50, 40)).toEqual({ x: 150, y: 140, w: 450, h: 360 })
  })

  it('stops at the minimum size without moving the anchor', () => {
    const got = resizeRect(r, 'w', 1000, 0)
    expect(got.w).toBe(MIN_W)
    expect(got.x + got.w).toBe(r.x + r.w)
    expect(resizeRect(r, 's', 0, -1000).h).toBe(MIN_H)
  })
})

describe('clampRect', () => {
  it('keeps the title bar below the menu bar', () => {
    expect(clampRect({ x: 10, y: -50, w: 300, h: 200 }, area).y).toBe(30)
  })

  it('keeps part of the window on screen horizontally', () => {
    expect(clampRect({ x: 5000, y: 100, w: 300, h: 200 }, area).x).toBe(920)
    expect(clampRect({ x: -5000, y: 100, w: 300, h: 200 }, area).x).toBe(-220)
  })

  it('shrinks windows larger than the work area', () => {
    const got = clampRect({ x: 0, y: 30, w: 4000, h: 4000 }, area)
    expect([got.w, got.h]).toEqual([1000, 600])
  })
})

describe('snap', () => {
  it.each([
    [500, 30, 'top'],
    [0, 300, 'left'],
    [999, 300, 'right'],
    [500, 300, null],
  ] as const)('snapZone(%d, %d) = %s', (x, y, want) => {
    expect(snapZone(x, y, area)).toBe(want)
  })

  it('splits the work area in halves', () => {
    expect(zoneRect('left', area)).toEqual({ x: 0, y: 30, w: 500, h: 600 })
    expect(zoneRect('right', area)).toEqual({ x: 500, y: 30, w: 500, h: 600 })
    expect(zoneRect('top', area)).toEqual(area)
  })
})

describe('fitRect', () => {
  const area = { x: 0, y: 30, w: 1000, h: 600 }

  it('keeps a window that already fits', () => {
    expect(fitRect({ x: 100, y: 100, w: 400, h: 300 }, area)).toEqual({
      x: 100,
      y: 100,
      w: 400,
      h: 300,
    })
  })

  it('moves a window back inside every edge', () => {
    expect(fitRect({ x: 800, y: 500, w: 400, h: 300 }, area)).toEqual({
      x: 600,
      y: 330,
      w: 400,
      h: 300,
    })
    expect(fitRect({ x: -50, y: 0, w: 400, h: 300 }, area)).toEqual({ x: 0, y: 30, w: 400, h: 300 })
  })

  it('shrinks a window bigger than the area', () => {
    expect(fitRect({ x: 10, y: 10, w: 1400, h: 900 }, area)).toEqual(area)
  })
})
