import { describe, expect, it } from 'vitest'

import { fitWithin, mixWallpaper, wallpaperMode } from './wallpaper'

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

describe('wallpaperMode', () => {
  const on = { wallpaper: true, wallpaperImage: true }

  it('paints the picture when there is one and the preset allows it', () => {
    expect(wallpaperMode(on, true)).toBe('image')
  })

  it('keeps the blobs when no picture was chosen', () => {
    expect(wallpaperMode(on, false)).toBe('blobs')
  })

  it('drops to a flat color when the preset turns pictures and blobs off', () => {
    expect(wallpaperMode({ wallpaper: false, wallpaperImage: false }, true)).toBe('flat')
  })

  it('falls back to the blobs when only the picture is switched off', () => {
    expect(wallpaperMode({ wallpaper: true, wallpaperImage: false }, true)).toBe('blobs')
  })
})

describe('fitWithin', () => {
  it('shrinks the long side to the limit and keeps the ratio', () => {
    expect(fitWithin(5120, 2880, 2560)).toEqual({ w: 2560, h: 1440 })
    expect(fitWithin(2000, 4000, 2560)).toEqual({ w: 1280, h: 2560 })
  })

  it('never enlarges', () => {
    expect(fitWithin(800, 600, 2560)).toEqual({ w: 800, h: 600 })
  })

  it('refuses an empty picture', () => {
    expect(fitWithin(0, 100, 2560)).toEqual({ w: 0, h: 0 })
    expect(fitWithin(Number.NaN, 100, 2560)).toEqual({ w: 0, h: 0 })
  })
})
