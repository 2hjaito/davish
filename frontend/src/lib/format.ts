import { locale } from './i18n'

const num = new Intl.NumberFormat(locale, { maximumFractionDigits: 1 })

const whole = new Intl.NumberFormat(locale, { maximumFractionDigits: 0 })

const UNITS = ['B', 'KB', 'MB', 'GB', 'TB'] as const

function scale(bytes: number): [number, (typeof UNITS)[number]] {
  let v = bytes
  let i = 0
  while (v >= 1024 && i < UNITS.length - 1) {
    v /= 1024
    i++
  }
  return [v, UNITS[i] ?? 'B']
}

export function formatBytes(bytes: number): string {
  const [v, unit] = scale(bytes)
  return `${num.format(v)} ${unit}`
}

export function formatRate(bps: number): string {
  return `${formatBytes(bps)}/s`
}

/** A rate in at most three digits, for tight spots like the menu bar. */
export function formatRateShort(bps: number): string {
  const [v, unit] = scale(bps)
  return `${(v < 100 ? num : whole).format(v)} ${unit}/s`
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

/** A stopwatch reading: "0:42", "12:05", "1:02:03". */
export function formatElapsed(ms: number): string {
  const s = Math.max(0, Math.floor(ms / 1000))
  const h = Math.floor(s / 3600)
  const mm = Math.floor((s % 3600) / 60)
  const ss = String(s % 60).padStart(2, '0')
  return h > 0 ? `${h}:${String(mm).padStart(2, '0')}:${ss}` : `${mm}:${ss}`
}
