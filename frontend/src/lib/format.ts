import { locale } from './i18n'

const num = new Intl.NumberFormat(locale, { maximumFractionDigits: 1 })

const UNITS = ['B', 'KB', 'MB', 'GB', 'TB'] as const

export function formatBytes(bytes: number): string {
  let v = bytes
  let i = 0
  while (v >= 1024 && i < UNITS.length - 1) {
    v /= 1024
    i++
  }
  return `${num.format(v)} ${UNITS[i]}`
}

export function formatRate(bps: number): string {
  return `${formatBytes(bps)}/s`
}

export function formatGB(bytes: number): string {
  return num.format(bytes / 1024 ** 3)
}

export function formatClock(d: Date, weekdays: readonly string[]): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  const day = weekdays[d.getDay()] ?? ''
  return `${day} ${pad(d.getDate())}/${pad(d.getMonth() + 1)}  ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

/** "23%", "2,5%": one decimal only below 10, where it still means something. */
export function formatPct(pct: number): string {
  const v = pct < 10 ? Math.round(pct * 10) / 10 : Math.round(pct)
  return `${num.format(v)}%`
}

export interface DurationUnits {
  day: string
  hour: string
  minute: string
}

/** The two largest units: "4 ngày 13 giờ", "3 giờ 12 phút", "5 phút". */
export function formatDuration(seconds: number, u: DurationUnits): string {
  const s = Math.max(0, Math.floor(seconds))
  const d = Math.floor(s / 86400)
  const h = Math.floor((s % 86400) / 3600)
  const m = Math.floor((s % 3600) / 60)
  if (d > 0) return h > 0 ? `${d} ${u.day} ${h} ${u.hour}` : `${d} ${u.day}`
  if (h > 0) return m > 0 ? `${h} ${u.hour} ${m} ${u.minute}` : `${h} ${u.hour}`
  return `${m} ${u.minute}`
}
