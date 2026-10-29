import { describe, expect, it } from 'vitest'

import { formatBytes, formatClock, formatDuration, formatGB, formatPct, formatRate } from './format'
import { t } from './i18n'

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

  it.each([
    [0, '0%'],
    [2.46, '2,5%'],
    [9.94, '9,9%'],
    [23.4, '23%'],
    [100, '100%'],
  ])('formatPct(%d) = %s', (n, want) => {
    expect(formatPct(n)).toBe(want)
  })

  it.each([
    [42, '0 phút'],
    [5 * 60, '5 phút'],
    [3 * 3600, '3 giờ'],
    [3 * 3600 + 12 * 60 + 9, '3 giờ 12 phút'],
    [4 * 86400 + 13 * 3600 + 60, '4 ngày 13 giờ'],
    [2 * 86400 + 59, '2 ngày'],
  ])('formatDuration(%d) = %s', (s, want) => {
    expect(formatDuration(s, t.units)).toBe(want)
  })
})
