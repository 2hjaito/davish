import { afterAll, beforeAll, describe as group, expect, it } from 'vitest'

import type { Schedule } from '../../lib/api/cron'
import { loadLang } from '../../lib/i18n'
import { describe } from './describe'

const range = (lo: number, hi: number, k = 1) =>
  Array.from({ length: Math.floor((hi - lo) / k) + 1 }, (_, i) => lo + i * k)

function sched(raw: string, f: Partial<Schedule> = {}): Schedule {
  return {
    raw,
    reboot: false,
    minute: range(0, 59),
    hour: range(0, 23),
    dom: range(1, 31),
    month: range(1, 12),
    dow: range(0, 6),
    domAny: true,
    dowAny: true,
    ...f,
  }
}

it.each([
  [sched('* * * * *'), 'Mỗi phút'],
  [sched('*/5 * * * *', { minute: range(0, 55, 5) }), 'Mỗi 5 phút'],
  [sched('17 * * * *', { minute: [17] }), 'Mỗi giờ vào phút 17'],
  [sched('0 */6 * * *', { minute: [0], hour: [0, 6, 12, 18] }), 'Mỗi 6 giờ vào phút 00'],
  [sched('0 3 * * *', { minute: [0], hour: [3] }), 'Mỗi ngày lúc 03:00'],
  [sched('0 9,18 * * *', { minute: [0], hour: [9, 18] }), 'Mỗi ngày lúc 09:00, 18:00'],
  [
    sched('30 9 * * 1-5', { minute: [30], hour: [9], dow: [1, 2, 3, 4, 5], dowAny: false }),
    'Thứ 2 đến thứ 6 lúc 09:30',
  ],
  [
    sched('47 6 * * 0', { minute: [47], hour: [6], dow: [0], dowAny: false }),
    'Mỗi Chủ nhật lúc 06:47',
  ],
  [
    sched('0 0 * * 0,6', { minute: [0], hour: [0], dow: [0, 6], dowAny: false }),
    'Thứ 7 và Chủ nhật lúc 00:00',
  ],
  [
    sched('0 0 1 * *', { minute: [0], hour: [0], dom: [1], domAny: false }),
    'Ngày 1 hàng tháng lúc 00:00',
  ],
  [
    sched('0 8 1,15 * *', { minute: [0], hour: [8], dom: [1, 15], domAny: false }),
    'Ngày 1 và 15 hàng tháng lúc 08:00',
  ],
  [
    sched('0 8 1 * 1', {
      minute: [0],
      hour: [8],
      dom: [1],
      dow: [1],
      domAny: false,
      dowAny: false,
    }),
    'Ngày 1 hàng tháng hoặc mỗi thứ 2 lúc 08:00',
  ],
  [
    sched('0 12 * 1,7 *', { minute: [0], hour: [12], month: [1, 7] }),
    'Mỗi ngày lúc 12:00 (tháng 1 và 7)',
  ],
  [
    sched('*/10 * * * 1-5', { minute: range(0, 50, 10), dow: [1, 2, 3, 4, 5], dowAny: false }),
    'Thứ 2 đến thứ 6 mỗi 10 phút',
  ],
  [sched('@reboot', { reboot: true }), 'Khi server khởi động'],
  [sched('1,2,9 3-20/7 * * *', { minute: [1, 2, 9], hour: [3, 10, 17] }), '1,2,9 3-20/7 * * *'],
])('%s', (s, want) => expect(describe(s)).toBe(want))

group('in English', () => {
  beforeAll(() => loadLang('en'))
  afterAll(() => loadLang('vi'))

  it.each([
    [sched('*/5 * * * *', { minute: range(0, 55, 5) }), 'Every 5 minutes'],
    [sched('0 9,18 * * *', { minute: [0], hour: [9, 18] }), 'Every day at 09:00, 18:00'],
    [
      sched('30 9 * * 1-5', { minute: [30], hour: [9], dow: [1, 2, 3, 4, 5], dowAny: false }),
      'Monday to Friday at 09:30',
    ],
    [
      sched('0 8 1,15 * *', { minute: [0], hour: [8], dom: [1, 15], domAny: false }),
      'On day 1 and 15 of the month at 08:00',
    ],
    [
      sched('0 12 * 1,7 *', { minute: [0], hour: [12], month: [1, 7] }),
      'Every day at 12:00 (months: 1 and 7)',
    ],
  ])('%s', (s, want) => expect(describe(s)).toBe(want))
})
