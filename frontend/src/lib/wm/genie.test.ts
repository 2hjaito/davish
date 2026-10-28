import { describe, expect, it } from 'vitest'

import { genieFrame } from './genie'

const win = { x: 100, y: 100, width: 400, height: 300 }
const tile = { x: 580, y: 700, width: 40, height: 40 }

/** The [x, y%] points of a polygon() clip-path. */
function points(clip: string): [number, number][] {
  return clip
    .slice('polygon('.length, -1)
    .split(', ')
    .map((p) => {
      const [x, y] = p.split(' ')
      return [parseFloat(x!), parseFloat(y!)]
    })
}

describe('genieFrame', () => {
  it('leaves the window as it is at the start', () => {
    const f = genieFrame(win, tile, 0)
    expect(f.transform).toBe('translateY(0px) scaleY(1)')
    expect(f.opacity).toBe(1)
    const xs = points(f.clipPath).map(([x]) => x)
    expect(Math.min(...xs)).toBe(0)
    expect(Math.max(...xs)).toBe(400)
  })

  it('ends exactly on the tile', () => {
    const f = genieFrame(win, tile, 1)
    // Moved down to the tile's top and squeezed to its height.
    expect(f.transform).toBe('translateY(600px) scaleY(0.1333)')
    expect(f.opacity).toBe(0)
    const xs = points(f.clipPath).map(([x]) => x)
    expect(Math.min(...xs)).toBe(480)
    expect(Math.max(...xs)).toBe(520)
  })

  it('narrows the bottom before the top', () => {
    const pts = points(genieFrame(win, tile, 0.4).clipPath)
    const topWidth = pts[pts.length - 1]![0] - pts[0]![0]
    const half = pts.length / 2
    const bottomWidth = pts[half]![0] - pts[half - 1]![0]
    expect(bottomWidth).toBeLessThan(topWidth / 2)
  })

  it('keeps the same number of points so frames interpolate', () => {
    const counts = [0, 0.3, 0.7, 1].map((p) => points(genieFrame(win, tile, p).clipPath).length)
    expect(new Set(counts).size).toBe(1)
  })
})
