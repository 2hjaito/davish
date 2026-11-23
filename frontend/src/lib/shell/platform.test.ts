import { describe, expect, it } from 'vitest'
import { isMac, shortcut } from './platform'

describe('shortcut', () => {
  it.skipIf(isMac)('spells out Mac modifier glyphs off the Mac', () => {
    expect(shortcut('K')).toBe('Ctrl+K')
    expect(shortcut('⇧U')).toBe('Ctrl+Shift+U')
    expect(shortcut('⌥L')).toBe('Ctrl+Alt+L')
  })
})
