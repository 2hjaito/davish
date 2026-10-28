// The genie effect of macOS: a window pours into its dock tile through a
// funnel with curved sides. A webview cannot warp content along a mesh, so
// this stretches the window vertically from the funnel's top down to the
// tile and clips it to the funnel's outline.

import type { Box } from './geometry'

/** Points along each side of the funnel; more is smoother and costlier. */
const ROWS = 12

export interface GenieFrame {
  transform: string
  clipPath: string
  opacity: number
}

const clamp = (v: number) => Math.min(1, Math.max(0, v))
const smooth = (v: number) => v * v * (3 - 2 * v)
const lerp = (a: number, b: number, t: number) => a + (b - a) * t

/**
 * The window's look at progress p: 0 is the window as it is, 1 is all of
 * it inside tile. First the bottom stretches down and narrows to the tile;
 * then the top follows it in. win and tile are in screen coordinates; the
 * transform needs transform-origin at the window's top left.
 */
export function genieFrame(win: Box, tile: Box, p: number): GenieFrame {
  const pour = smooth(clamp(p / 0.45))
  const slide = smooth(clamp((p - 0.3) / 0.7))

  const top = lerp(win.y, tile.y, slide)
  const bottom = lerp(win.y + win.height, tile.y + tile.height, pour)
  const height = Math.max(bottom - top, 0.5)

  // Edges of the funnel at its top and bottom, in window coordinates.
  const tileLeft = tile.x - win.x
  const tileRight = tile.x + tile.width - win.x
  const topLeft = lerp(0, tileLeft, slide)
  const topRight = lerp(win.width, tileRight, slide)
  const bottomLeft = lerp(0, tileLeft, pour)
  const bottomRight = lerp(win.width, tileRight, pour)

  const left: string[] = []
  const right: string[] = []
  for (let i = 0; i <= ROWS; i++) {
    const v = i / ROWS
    const bend = smooth(v)
    const y = round(v * 100)
    left.push(`${round(lerp(topLeft, bottomLeft, bend))}px ${y}%`)
    right.push(`${round(lerp(topRight, bottomRight, bend))}px ${y}%`)
  }
  return {
    transform: `translateY(${round(top - win.y)}px) scaleY(${round(height / win.height, 4)})`,
    clipPath: `polygon(${[...left, ...right.reverse()].join(', ')})`,
    // Fades only at the very end, as the window vanishes into the tile.
    opacity: round(1 - smooth(clamp((p - 0.8) / 0.2)), 3),
  }
}

function round(v: number, digits = 2): number {
  const f = 10 ** digits
  return Math.round(v * f) / f
}

/** genieFrame as a CSS declaration list, for Svelte transitions. */
export function genieCss(win: Box, tile: Box, p: number): string {
  const f = genieFrame(win, tile, p)
  return `transform-origin: 0 0; transform: ${f.transform}; clip-path: ${f.clipPath}; opacity: ${f.opacity}`
}
