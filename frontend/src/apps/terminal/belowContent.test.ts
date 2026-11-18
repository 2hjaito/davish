import { describe, expect, it } from 'vitest'

import { belowContent, type BufferLike } from './belowContent'

function buffer(lines: string[], baseY = 0): BufferLike {
  return {
    length: lines.length,
    baseY,
    getLine: (y) => {
      const line = lines.at(y)
      return line === undefined ? undefined : { translateToString: () => line.trimEnd() }
    },
  }
}

describe('belowContent', () => {
  it('goes to the last line with text, not where the cursor was left', () => {
    const screen = [
      '=> [builder 3/6] COPY',
      '=> [builder 4/6] RUN npm ci',
      '=> [builder 5/6] COPY',
      '',
      '',
    ]
    expect(belowContent(buffer(screen))).toBe('\x1b[3;1H')
  })

  it('counts rows from the top of the viewport', () => {
    const lines = ['old', 'old', 'old', 'shown', 'last line', '   ']
    expect(belowContent(buffer(lines, 3))).toBe('\x1b[2;1H')
  })

  it('starts at the top of an empty viewport', () => {
    expect(belowContent(buffer(['scrollback', '', ''], 1))).toBe('\x1b[H')
    expect(belowContent(buffer([]))).toBe('\x1b[H')
  })
})
