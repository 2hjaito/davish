import { describe, expect, it } from 'vitest'

import { mixWallpaper } from './wallpaper'

describe('mixWallpaper', () => {
  it('uses the swatch itself when the mix is 100%', () => {
    expect(mixWallpaper('#8fb2ff', '#cdd7ea', 100)).toBe('#8fb2ff')
    expect(mixWallpaper('#ffb38a', '#cdd7ea', 100)).toBe('#ffb38a')
  })

  it('dims the swatch into the dark base', () => {
    expect(mixWallpaper('#8fb2ff', '#161b27', 0)).toBe('#161b27')
    expect(mixWallpaper('#8fb2ff', '#161b27', 38)).toBe('#445479')
  })

  it('rejects a color that is not hex', () => {
    expect(mixWallpaper('blue', '#161b27', 38)).toBeNull()
  })
})
