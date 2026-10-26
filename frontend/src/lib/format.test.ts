import { describe, expect, it } from 'vitest'

import { formatBytes, formatClock, formatGB, formatRate } from './format'
import { t } from './i18n/vi'

describe('format', () => {
  it.each([
    [0, '0 B'],
    [210, '210 B'],
    [2.4 * 1024, '2,4 KB'],
    [1.2 * 1024 ** 2, '1,2 MB'],
  ])('formatBytes(%d) = %s', (n, want) => {
    expect(formatBytes(n)).toBe(want)
  })

  it('formats rates and GB', () => {
    expect(formatRate(1.2 * 1024 ** 2)).toBe('1,2 MB/s')
    expect(formatGB(5.1 * 1024 ** 3)).toBe('5,1')
  })

  it('formats the menu bar clock', () => {
    expect(formatClock(new Date(2026, 9, 26, 21, 54), t.menubar.weekdays)).toBe('Th 2 26/10  21:54')
  })
})
