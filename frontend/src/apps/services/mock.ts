// Demo data for the plain browser preview (vite dev), where there is no Go side.
import type { ActionPlan, Journal, ServiceAction, Unit, UnitStatus } from '../../lib/api/services'

const units: Unit[] = [
  ['cron', 'active', 'running', 'Regular background program processing daemon'],
  ['davish-crash', 'failed', 'failed', 'Davish test service that always fails'],
  ['dbus', 'active', 'running', 'D-Bus System Message Bus'],
  ['nginx', 'active', 'running', 'A high performance web server and a reverse proxy server'],
  ['postgresql', 'active', 'exited', 'PostgreSQL RDBMS'],
  ['redis-server', 'inactive', 'dead', 'Advanced key-value store'],
  ['ssh', 'active', 'running', 'OpenBSD Secure Shell server'],
  ['systemd-journald', 'active', 'running', 'Journal Service'],
].map(([name, active, sub, description]) => ({
  name: `${name}.service`,
  load: 'loaded',
  active: active!,
  sub: sub!,
  description: description!,
}))

const delay = <T>(v: T) => new Promise<T>((r) => setTimeout(() => r(v), 150))

export const mockList = () => delay(units.map((u) => ({ ...u })))

export function mockStatus(name: string): Promise<UnitStatus> {
  const u = units.find((x) => x.name === name) ?? units[0]!
  const running = u.active === 'active'
  return delay({
    unit: u.name,
    description: u.description,
    load: u.load,
    active: u.active,
    sub: u.sub,
    enabled: u.name.startsWith('redis') ? 'disabled' : 'enabled',
    activeFor: running ? 3 * 86400 + 4 * 3600 : 0,
    mainPID: running ? 812 : 0,
    memory: running ? 18_400_000 : -1,
    restarts: 0,
    result: u.active === 'failed' ? 'exit-code' : 'success',
    file: `/usr/lib/systemd/system/${u.name}`,
  })
}

export function mockJournal(name: string): Promise<Journal> {
  const failed = units.find((x) => x.name === name)?.active === 'failed'
  const lines = [
    `2026-10-27T08:00:01+07:00 vps systemd[1]: Starting ${name}...`,
    failed
      ? '2026-10-27T08:00:01+07:00 vps sh[34]: error: cannot open /etc/nginx/nginx.conf'
      : `2026-10-27T08:00:01+07:00 vps systemd[1]: Started ${name}.`,
  ]
  if (failed)
    lines.push(`2026-10-27T08:00:01+07:00 vps systemd[1]: ${name}: Failed with result 'exit-code'.`)
  return delay({
    lines,
    restricted: false,
    sudo: `sudo journalctl -u ${name} -n 200 --no-pager -o short-iso`,
  })
}

export function mockPlan(unit: string, action: ServiceAction): Promise<ActionPlan> {
  const command = `systemctl ${action} ${unit}`
  return delay({ unit, action, command, confirm: action !== 'start', sudo: `sudo ${command}` })
}

/** Acts on the demo list; nginx refuses, to show the root path. */
export async function mockAct(unit: string, action: ServiceAction): Promise<void> {
  await delay(null)
  if (unit === 'nginx.service') throw { code: 'CMD_NEEDS_ROOT', message: 'needs root' }
  const u = units.find((x) => x.name === unit)
  if (!u) return
  ;[u.active, u.sub] = action === 'stop' ? ['inactive', 'dead'] : ['active', 'running']
}
