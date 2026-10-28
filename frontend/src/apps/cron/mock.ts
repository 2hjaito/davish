// Demo data for the plain browser preview (vite dev), where there is no Go side.
import type {
  CronChange,
  CronJob,
  CronPlan,
  CronPreview,
  CronRuns,
  CronTable,
  Schedule,
} from '../../lib/api/cron'

const range = (lo: number, hi: number, k = 1) =>
  Array.from({ length: Math.floor((hi - lo) / k) + 1 }, (_, i) => lo + i * k)

// A tiny parser for the demo: "m h * * *", every n minutes, @reboot, @daily.
function parse(raw: string): Schedule | null {
  const all: Schedule = {
    raw,
    reboot: false,
    minute: range(0, 59),
    hour: range(0, 23),
    dom: range(1, 31),
    month: range(1, 12),
    dow: range(0, 6),
    domAny: true,
    dowAny: true,
  }
  if (raw === '@reboot') return { ...all, reboot: true }
  if (raw === '@daily') return { ...all, minute: [0], hour: [0] }
  const f = raw.trim().split(/\s+/)
  if (f.length !== 5 || f.slice(2).some((x) => x !== '*')) return null
  const step = /^\*\/(\d+)$/.exec(f[0]!)
  if (step && f[1] === '*') return { ...all, minute: range(0, 59, Number(step[1])) }
  const m = Number(f[0])
  const h = f[1] === '*' ? null : Number(f[1])
  if (!Number.isInteger(m) || m < 0 || m > 59) return null
  if (h !== null && (!Number.isInteger(h) || h < 0 || h > 23)) return null
  return { ...all, minute: [m], hour: h === null ? range(0, 23) : [h] }
}

function nextRuns(s: Schedule): number[] {
  if (s.reboot) return []
  const out: number[] = []
  const t = new Date()
  t.setSeconds(0, 0)
  for (let i = 0; i < 60 * 24 * 8 && out.length < 3; i++) {
    t.setMinutes(t.getMinutes() + 1)
    if (s.minute.includes(t.getMinutes()) && s.hour.includes(t.getHours()))
      out.push(t.getTime() / 1000)
  }
  return out
}

let lines: [string, string, boolean][] = [
  ['0 3 * * *', '/usr/local/bin/backup-db.sh >> /var/log/backup.log 2>&1', true],
  ['*/5 * * * *', 'curl -fsS https://api.davitek.com/health > /dev/null', true],
  ['@reboot', '/home/deploy/start-worker.sh', true],
  ['30 2 * * *', '/usr/local/bin/rotate-old.sh', false],
]

const system: CronJob[] = [
  ['17 * * * *', 'cd / && run-parts --report /etc/cron.hourly', 18],
  [
    '25 6 * * *',
    'test -x /usr/sbin/anacron || { cd / && run-parts --report /etc/cron.daily; }',
    19,
  ],
].map(([raw, command, line]) => {
  const s = parse(raw as string)!
  return {
    id: `/etc/crontab:${line}`,
    source: '/etc/crontab',
    line: line as number,
    schedule: s,
    user: 'root',
    command: command as string,
    enabled: true,
    next: nextRuns(s),
    editable: false,
  }
})

const delay = <T>(v: T) => new Promise<T>((r) => setTimeout(() => r(v), 150))
const fail = (code: string) => Promise.reject({ code, message: code })

export function mockList(): Promise<CronTable> {
  const mine: CronJob[] = lines.map(([raw, command, enabled], i) => {
    const s = parse(raw)!
    return {
      id: `user:${i + 1}`,
      source: 'user',
      line: i + 1,
      schedule: s,
      command,
      enabled,
      next: nextRuns(s),
      editable: true,
    }
  })
  return delay({
    jobs: [...mine, ...system],
    env: [{ source: 'user', name: 'MAILTO', value: 'ops@davitek.com' }],
    zone: 'Asia/Ho_Chi_Minh',
    unreadable: [],
    version: 'demo',
  })
}

export function mockPreview(raw: string): Promise<CronPreview> {
  const s = parse(raw)
  return s ? delay({ schedule: s, next: nextRuns(s) }) : fail('INVALID_INPUT')
}

const index = (id: string) => Number(id.split(':')[1]) - 1

function render(i: number) {
  const [raw, command, enabled] = lines[i]!
  return `${enabled ? '' : '#'}${raw} ${command}`
}

export function mockPlan(c: CronChange): Promise<CronPlan> {
  if ((c.op === 'add' || c.op === 'edit') && !parse(c.schedule)) return fail('INVALID_INPUT')
  const after =
    c.op === 'add' || c.op === 'edit'
      ? `${c.schedule} ${c.command}`
      : c.op === 'delete'
        ? ''
        : `${c.op === 'disable' ? '#' : ''}${lines[index(c.id)]![0]} ${lines[index(c.id)]![1]}`
  return delay({
    command: 'crontab -l > /home/dangth/.crontab.bak-20261027T081005Z && crontab -',
    before: c.op === 'add' ? '' : render(index(c.id)),
    after,
    backup: '/home/dangth/.crontab.bak-20261027T081005Z',
    version: 'demo',
  })
}

export async function mockApply(c: CronChange): Promise<void> {
  await delay(null)
  const i = index(c.id)
  switch (c.op) {
    case 'add':
      lines = [...lines, [c.schedule, c.command, true]]
      break
    case 'edit':
      lines[i] = [c.schedule, c.command, lines[i]![2]]
      break
    case 'delete':
      lines = lines.filter((_, j) => j !== i)
      break
    default:
      lines[i]![2] = c.op === 'enable'
  }
}

export const mockRuns = (command: string): Promise<CronRuns> =>
  delay({
    lines: [3, 2, 1].map(
      (d) => `2026-10-${27 - d}T03:00:01+07:00 vps CRON[${900 + d}]: (dangth) CMD (${command})`,
    ),
    restricted: false,
    sudo: 'sudo journalctl -t CRON -t CROND -n 5000 --no-pager -o short-iso',
  })
