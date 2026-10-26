const num = new Intl.NumberFormat('vi-VN', { maximumFractionDigits: 1 })

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
