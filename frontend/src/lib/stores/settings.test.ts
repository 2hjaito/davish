import { describe, expect, it } from 'vitest'

import { clampFontSize, resolveTheme, TERMINAL_FONT } from './settings.svelte'

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
