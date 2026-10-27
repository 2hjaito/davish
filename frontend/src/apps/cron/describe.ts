// Says when a cron schedule runs, in Vietnamese: "Mỗi ngày lúc 03:00".
import type { Schedule } from '../../lib/api/cron'
import { t } from '../../lib/i18n/vi'

const pad = (n: number) => String(n).padStart(2, '0')

/** The step of an arithmetic run that starts at lo and fills the field, or 0. */
function step(values: readonly number[], lo: number, size: number): number {
  if (values.length < 2 || values[0] !== lo) return 0
  const k = values[1]! - values[0]!
  if (values.some((v, i) => v !== lo + i * k)) return 0
  // The run must reach the end of the field, as */k does.
  return values[values.length - 1]! + k >= lo + size ? k : 0
}

function list(values: readonly number[], name: (v: number) => string): string {
  const names = values.map(name)
  return names.length <= 1
    ? (names[0] ?? '')
    : `${names.slice(0, -1).join(', ')} ${t.cron.and} ${names[names.length - 1]}`
}

function timePart(s: Schedule): string | null {
  const everyMinute = s.minute.length === 60
  const everyHour = s.hour.length === 24
  if (everyMinute && everyHour) return t.cron.everyMinute
  const m = step(s.minute, 0, 60)
  if (m > 1 && everyHour) return t.cron.everyNMinutes(m)
  if (s.minute.length !== 1) return null
  const mm = s.minute[0]!
  if (everyHour) return t.cron.everyHourAt(pad(mm))
  const h = step(s.hour, 0, 24)
  if (h > 1) return t.cron.everyNHoursAt(h, pad(mm))
  if (s.hour.length <= 4) return t.cron.at(s.hour.map((hh) => `${pad(hh)}:${pad(mm)}`).join(', '))
  return null
}

function dayPart(s: Schedule): string | null {
  const allDom = s.domAny || s.dom.length === 31
  const allDow = s.dowAny || s.dow.length === 7
  const weekdays = s.dow.join() === '1,2,3,4,5'
  const weekend = s.dow.join() === '0,6'
  const dows = weekdays
    ? t.cron.weekdays
    : weekend
      ? t.cron.weekend
      : t.cron.onDays(list(s.dow, (d) => t.cron.dayNames[d]!))
  const doms = t.cron.monthDays(list(s.dom, String))
  if (allDom && allDow) return null
  if (allDom) return dows
  if (allDow) return doms
  return t.cron.either(doms, dows)
}

/** "Mỗi ngày lúc 03:00", "Thứ 2 đến thứ 6 lúc 09:30"; the raw schedule when it does not fit a sentence. */
export function describe(s: Schedule): string {
  if (s.reboot) return t.cron.atBoot
  const time = timePart(s)
  if (time === null) return s.raw
  const day = dayPart(s)
  const months = s.month.length === 12 ? '' : ` ${t.cron.inMonths(list(s.month, String))}`
  const daily = time.startsWith(t.cron.atWord) && day === null
  const sentence = [day ?? (daily ? t.cron.everyDay : ''), time].filter(Boolean).join(' ') + months
  return sentence.charAt(0).toUpperCase() + sentence.slice(1)
}
