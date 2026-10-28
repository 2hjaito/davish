// Demo data for the plain browser preview (vite dev), where there is no Go side.
import type {
  ActionPlan,
  Container,
  ContainerAction,
  ContainerDetail,
  ContainerStats,
  LogLine,
} from '../../lib/api/docker'

const MB = 1024 ** 2

const list: Container[] = [
  ['api', 'myapp/api:2.4.1', 'running', 'Up 3 days', '0.0.0.0:8080->8080/tcp'],
  ['db', 'postgres:16', 'running', 'Up 3 days', '5432/tcp'],
  ['migrate', 'myapp/api:2.4.1', 'exited', 'Exited (0) 3 days ago', ''],
  ['redis', 'redis:7-alpine', 'running', 'Up 3 days', '6379/tcp'],
  ['worker', 'myapp/worker:2.4.1', 'restarting', 'Restarting (1) 4 seconds ago', ''],
].map(([name, image, state, status, ports]) => ({
  id: `${name}0123456789abcdef`,
  name: name!,
  image: image!,
  state: state!,
  status: status!,
  ports: ports!,
  command: 'docker-entrypoint.sh',
  created: '2026-10-24 08:00:00 +0700 +07',
}))

const delay = <T>(v: T, ms = 150) => new Promise<T>((r) => setTimeout(() => r(v), ms))

export const mockList = () => delay(list.map((c) => ({ ...c })))

export const mockStats = (): Promise<ContainerStats[]> =>
  delay(
    list
      .filter((c) => c.state === 'running')
      .map((c, i) => ({
        id: c.id,
        cpu: [2.4, 0.8, 0.3][i] ?? 0.1,
        memUsed: [182 * MB, 96 * MB, 8 * MB][i] ?? MB,
        memLimit: 7.7 * 1024 * MB,
        pids: 12,
      })),
    600,
  )

export function mockInspect(name: string): Promise<ContainerDetail> {
  const c = list.find((x) => x.name === name) ?? list[0]!
  const started = new Date(Date.now() - (c.state === 'running' ? 3 * 86400e3 : 4e3)).toISOString()
  const detail = {
    name: c.name,
    image: c.image,
    state: c.state,
    exitCode: c.state === 'restarting' ? 1 : 0,
    error: '',
    oomKilled: false,
    health: c.name === 'api' ? 'healthy' : '',
    startedAt: started,
    restarts: c.state === 'restarting' ? 37 : 0,
    policy: c.name === 'migrate' ? 'no' : 'unless-stopped',
    ports: c.ports ? [c.ports.replace('->', '→')] : [],
    mounts: c.name === 'db' ? ['/srv/pgdata → /var/lib/postgresql/data'] : [],
  }
  return delay({ ...detail, json: JSON.stringify({ Name: `/${c.name}`, ...detail }, null, 2) })
}

export function mockLogs(name: string): Promise<LogLine[]> {
  const at = (s: number) => new Date(Date.now() - s * 1000).toISOString()
  if (name === 'worker')
    return delay(
      [12, 8, 4].map((s) => ({
        time: at(s),
        text: 'Error: cannot open /app/config/worker.yml: no such file or directory',
        stderr: true,
      })),
    )
  return delay([
    { time: at(30), text: 'Listening on :8080', stderr: false },
    { time: at(20), text: 'GET /health 200 1.2ms', stderr: false },
    { time: at(10), text: 'WARN slow query took 1200ms', stderr: true },
  ])
}

export function mockPlan(name: string, action: ContainerAction): Promise<ActionPlan> {
  const command = `docker ${action} ${name}`
  return delay({ name, action, command, confirm: action !== 'start', sudo: `sudo ${command}` })
}

/** Acts on the demo list; db refuses, to show the root path. */
export async function mockAct(name: string, action: ContainerAction): Promise<void> {
  await delay(null)
  if (name === 'db') throw { code: 'CMD_NEEDS_ROOT', message: 'needs root' }
  const c = list.find((x) => x.name === name)
  if (c) c.state = action === 'stop' ? 'exited' : 'running'
}
