// Run times in the server's zone, as cron sees them.
import { t } from '../../lib/i18n/vi'

/** "Hôm nay, 14:15", "Ngày mai, 03:00", "Th 4, 29/10 03:00", in zone. */
export function formatRun(unix: number, zone: string, now = Date.now()): string {
  const parts = (ms: number) => {
    const p = new Intl.DateTimeFormat('en-GB', {
      timeZone: zone,
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      weekday: 'short',
      hourCycle: 'h23',
    }).formatToParts(ms)
    const get = (type: string) => p.find((x) => x.type === type)?.value ?? ''
    return {
      date: `${get('year')}-${get('month')}-${get('day')}`,
      day: get('day'),
      month: get('month'),
      time: `${get('hour')}:${get('minute')}`,
      weekday: ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'].indexOf(get('weekday')),
    }
  }
  const at = parts(unix * 1000)
  if (at.date === parts(now).date) return t.cron.today(at.time)
  if (at.date === parts(now + 86_400_000).date) return t.cron.tomorrow(at.time)
  return `${t.cron.weekdayShort[at.weekday] ?? ''}, ${at.day}/${at.month} ${at.time}`
}
