import { describe, expect, it } from 'vitest'

import { luminance, meanLuminance, toneOf, TONE_THRESHOLD } from './wallpaperTone'

describe('luminance', () => {
  it('is 0 for black and 1 for white', () => {
    expect(luminance(0, 0, 0)).toBe(0)
    expect(luminance(255, 255, 255)).toBeCloseTo(1, 5)
  })

  it('weighs green most and blue least, as the eye does', () => {
    expect(luminance(0, 255, 0)).toBeGreaterThan(luminance(255, 0, 0))
    expect(luminance(255, 0, 0)).toBeGreaterThan(luminance(0, 0, 255))
  })
})

describe('meanLuminance', () => {
  it('averages the pixels and ignores alpha', () => {
    // One black and one white pixel.
    const mean = meanLuminance([0, 0, 0, 255, 255, 255, 255, 0])
    expect(mean).toBeCloseTo(0.5, 5)
  })

  it('says middle for no pixels, so nothing is guessed from nothing', () => {
    expect(meanLuminance([])).toBe(0.5)
  })
})

describe('toneOf', () => {
  it('is dark for a dark picture and light for a bright one', () => {
    expect(toneOf(0.05)).toBe('dark')
    expect(toneOf(0.6)).toBe('light')
  })

  it('switches at the threshold', () => {
    expect(toneOf(TONE_THRESHOLD - 0.001)).toBe('dark')
    expect(toneOf(TONE_THRESHOLD)).toBe('light')
  })

  it('counts the dim layer: a bright picture dimmed enough is dark', () => {
    expect(toneOf(0.5)).toBe('light')
    expect(toneOf(0.5, 70)).toBe('dark')
  })

  it('keeps junk dim levels in range', () => {
    expect(toneOf(0.5, -20)).toBe('light')
    expect(toneOf(0.5, 500)).toBe('dark')
  })
})
