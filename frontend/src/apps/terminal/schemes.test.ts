import { describe, expect, it } from 'vitest'

import { TERMINAL_SCHEMES } from '../../lib/stores/settings.svelte'
import { schemeFor, SCHEMES, schemeTheme, swatch } from './schemes'

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
