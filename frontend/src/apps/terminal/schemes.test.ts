import { describe, expect, it } from 'vitest'

import { TERMINAL_SCHEMES } from '../../lib/stores/settings.svelte'
import { DAVISH_ANSI, schemeFor, SCHEMES, schemeTheme, swatch } from './schemes'

const COLOR = /^(#[0-9a-f]{6}|rgba\(\d+, \d+, \d+, [\d.]+\))$/i

describe('schemes', () => {
  it('has colors for every scheme but Davish, which comes from the tokens', () => {
    expect(schemeFor('davish')).toBeUndefined()
    for (const id of TERMINAL_SCHEMES.filter((s) => s !== 'davish')) {
      expect(schemeFor(id)).toBe(SCHEMES[id as keyof typeof SCHEMES])
    }
  })

  it.each(Object.entries(SCHEMES))('%s: every color is a hex or rgba', (_id, s) => {
    expect(s.ansi).toHaveLength(16)
    for (const c of [s.background, s.foreground, s.cursor, s.selection, ...s.ansi]) {
      expect(c).toMatch(COLOR)
    }
  })

  it('draws on its background when solid and on the glass when not', () => {
    const s = SCHEMES.dracula
    expect(schemeTheme(s, true).background).toBe(s.background)
    expect(schemeTheme(s, false).background).toBe('#00000000')
    expect(schemeTheme(s, true).brightWhite).toBe(s.ansi[15])
    expect(schemeTheme(s, true).red).toBe(s.ansi[1])
  })

  it('gives a tile its background, text and three accents', () => {
    for (const id of TERMINAL_SCHEMES) {
      const sw = swatch(id)
      expect(sw.accents).toHaveLength(3)
      expect(sw.bg).toMatch(COLOR)
      expect(sw.fg).toMatch(COLOR)
    }
  })
})

describe('the Davish ANSI colors', () => {
  const lum = (hex: string) => {
    const [r, g, b] = [1, 3, 5].map((i) => {
      const c = Number.parseInt(hex.slice(i, i + 2), 16) / 255
      return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4
    }) as [number, number, number]
    return 0.2126 * r + 0.7152 * g + 0.0722 * b
  }
  const contrast = (a: string, b: string) => {
    const [hi, lo] = [lum(a), lum(b)].sort((x, y) => y - x) as [number, number]
    return (hi + 0.05) / (lo + 0.05)
  }
  const BG = '#1d1f25' // --term-bg-solid

  it('keeps the grey of stack traces and hints readable on the dark background', () => {
    expect(contrast(DAVISH_ANSI.brightBlack, BG)).toBeGreaterThanOrEqual(7)
  })

  it('keeps every color legible, even plain black (used for dim text)', () => {
    for (const [name, hex] of Object.entries(DAVISH_ANSI))
      expect(contrast(hex, BG), name).toBeGreaterThanOrEqual(3.5)
  })

  it('keeps brights at least as light as their normal color', () => {
    expect(lum(DAVISH_ANSI.brightBlack)).toBeGreaterThan(lum(DAVISH_ANSI.black))
    expect(lum(DAVISH_ANSI.brightRed)).toBeGreaterThan(lum(DAVISH_ANSI.red))
  })
})
