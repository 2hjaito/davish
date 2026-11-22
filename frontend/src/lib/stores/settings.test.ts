import { describe, expect, it } from 'vitest'

import {
  clampFontSize,
  clampScrollback,
  resolveTheme,
  settings,
  TERMINAL_FONT,
  TERMINAL_SCROLLBACK,
  WALLPAPER_ACCENTS,
} from './settings.svelte'

describe('clampFontSize', () => {
  it.each([
    [12.5, 12.5],
    [14, 14],
    [13.2, 13],
    [13.3, 13.5],
    [4, TERMINAL_FONT.min],
    [99, TERMINAL_FONT.max],
    [Number.NaN, TERMINAL_FONT.default],
  ])('clampFontSize(%d) = %d', (size, want) => {
    expect(clampFontSize(size)).toBe(want)
  })

  it('falls back to the default for a non-number from storage', () => {
    expect(clampFontSize('big' as unknown as number)).toBe(TERMINAL_FONT.default)
  })
})

describe('resolveTheme', () => {
  it.each([
    ['system', false, 'light'],
    ['system', true, 'dark'],
    ['light', true, 'light'],
    ['dark', false, 'dark'],
  ] as const)('resolveTheme(%s, systemDark=%s) = %s', (theme, systemDark, want) => {
    expect(resolveTheme(theme, systemDark)).toBe(want)
  })
})

describe('wallpaperAccent', () => {
  it('keeps any #rrggbb color, lowercased', () => {
    settings.set('wallpaperAccent', '#12AB9F')
    expect(settings.wallpaperAccent).toBe('#12ab9f')
  })

  it('falls back to the first swatch for anything else', () => {
    for (const bad of ['red', '#fff', 'url(x)', '']) {
      settings.set('wallpaperAccent', bad)
      expect(settings.wallpaperAccent).toBe(WALLPAPER_ACCENTS[0])
    }
  })
})

describe('clampScrollback', () => {
  it.each([...TERMINAL_SCROLLBACK])('keeps the choice %d', (n) => {
    expect(clampScrollback(n)).toBe(n)
  })

  it.each([0, 12, 999999, -1, Number.NaN, '5000', null, undefined])(
    'falls back to 5000 for %s',
    (junk) => {
      expect(clampScrollback(junk)).toBe(5000)
    },
  )
})

describe('terminalOpaque', () => {
  it('is off by default, and on with the setting', () => {
    settings.usePreset('beautiful')
    settings.set('terminalGpu', false)
    settings.set('terminalSolid', false)
    expect(settings.terminalOpaque).toBe(false)
    settings.set('terminalSolid', true)
    expect(settings.terminalOpaque).toBe(true)
  })

  it('follows GPU drawing and the solid fills of the performance setting', () => {
    settings.usePreset('beautiful')
    settings.set('terminalSolid', false)
    settings.set('terminalGpu', true)
    expect(settings.terminalOpaque).toBe(true)
    settings.set('terminalGpu', false)
    settings.usePreset('light')
    expect(settings.terminalOpaque).toBe(true)
    settings.usePreset('beautiful')
    expect(settings.terminalOpaque).toBe(false)
  })
})
