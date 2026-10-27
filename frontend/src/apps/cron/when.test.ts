import { expect, it } from 'vitest'

import { formatRun } from './when'

it('formats runs in the server zone, relative to its today', () => {
  // 2026-10-27 07:07 UTC is 14:07 in Ho Chi Minh City, a Tuesday.
  const now = Date.UTC(2026, 9, 27, 7, 7)
  const z = 'Asia/Ho_Chi_Minh'
  expect(formatRun(Date.UTC(2026, 9, 27, 8, 0) / 1000, z, now)).toBe('Hôm nay, 15:00')
  expect(formatRun(Date.UTC(2026, 9, 27, 20, 0) / 1000, z, now)).toBe('Ngày mai, 03:00')
  expect(formatRun(Date.UTC(2026, 9, 29, 2, 30) / 1000, z, now)).toBe('Th 5, 29/10 09:30')
  // The same instant is still "today" in UTC.
  expect(formatRun(Date.UTC(2026, 9, 27, 20, 0) / 1000, 'UTC', now)).toBe('Hôm nay, 20:00')
})
