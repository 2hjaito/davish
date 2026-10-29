// Demo data for the plain browser preview (vite dev), where there is no Go side.
import type {
  ActionPlan,
  Container,
  ContainerAction,
  ContainerDetail,
  ContainerStats,
  Image,
  ImageDetail,
  ImagePlan,
  LogLine,
  Volume,
  VolumePlan,
} from '../../lib/api/docker'

const MB = 1024 ** 2

// A compose project (shop) beside containers started by hand.
const list: Container[] = (
  [
    [
      'shop-api-1',
      'myapp/api:2.4.1',
      'running',
      'Up 3 days',
      '0.0.0.0:8080->8080/tcp, :::8080->8080/tcp',
      '3 days ago',
      'shop',
      'api',
    ],
    ['shop-db-1', 'postgres:16', 'running', 'Up 3 days', '5432/tcp', '3 days ago', 'shop', 'db'],
    [
      'shop-redis-1',
      'redis:7-alpine',
      'running',
      'Up 3 days',
      '6379/tcp',
      '3 days ago',
      'shop',
      'redis',
    ],
    [
      'shop-worker-1',
      'myapp/worker:2.4.1',
      'restarting',
      'Restarting (1) 4 seconds ago',
      '',
      '3 days ago',
      'shop',
      'worker',
    ],
    ['migrate', 'myapp/api:2.4.1', 'exited', 'Exited (0) 3 days ago', '', '3 days ago', '', ''],
    [
      'zalo-oa-gateway',
      'davitek/zalo-gateway:latest',
      'running',
      'Up 26 hours',
      '127.0.0.1:3000->3000/tcp',
      '4 weeks ago',
      '',
      '',
    ],
  ] as const
).map(([name, image, state, status, ports, runningFor, project, service], i) => ({
  id: `${(i + 10).toString(16)}cf41f40809010123456789abcdef`,
  name,
  image,
  state,
  status,
  ports,
  command: 'docker-entrypoint.sh',
  created: '2026-10-24 08:00:00 +0700 +07',
  runningFor,
  project: project || undefined,
  service: service || undefined,
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
  if (name === 'shop-db-1') throw { code: 'CMD_NEEDS_ROOT', message: 'needs root' }
  const i = list.findIndex((x) => x.name === name)
  const c = list[i]
  if (!c) return
  if (action === 'rm') list.splice(i, 1)
  else c.state = action === 'stop' ? 'exited' : 'running'
}

let images: Image[] = [
  ['sha256:cf41f4080901', 'myapp/api', '2.4.1', '2 days ago', 431 * MB, 2],
  ['sha256:9a68dc314d7a', 'myapp/worker', '2.4.1', '2 days ago', 2140 * MB, 1],
  ['sha256:c53852c00c23', 'postgres', '16', '12 days ago', 1120 * MB, 1],
  ['sha256:acc6678fb1fa', 'redis', '7-alpine', '17 days ago', 41 * MB, 1],
  ['sha256:4731c9d4264a', 'myapp/api', '2.3.0', '3 weeks ago', 419 * MB, 0],
  ['sha256:2703a43d0d47', '<none>', '<none>', '1 month ago', 434 * MB, 0],
].map(([id, repository, tag, createdSince, size, containers]) => ({
  id: id as string,
  repository: repository as string,
  tag: tag as string,
  created: '2026-10-26 09:12:40 +0700 +07',
  createdSince: createdSince as string,
  size: size as number,
  containers: containers as number,
  dangling: repository === '<none>',
}))

export const mockImages = () => delay(images.map((i) => ({ ...i })))

export const mockImage = (_ref: string): Promise<ImageDetail> =>
  delay({
    layers: [
      { createdBy: 'ADD alpine-minirootfs-3.20.3-aarch64.tar.gz / # buildkit', size: 9.31 * MB },
      { createdBy: 'CMD ["/bin/sh"]', size: 0 },
      { createdBy: 'ENV NODE_VERSION=22.23.4', size: 0 },
      {
        createdBy: 'RUN addgroup -g 1000 node && adduser -u 1000 -G node -s /bin/sh -D node',
        size: 158.33 * MB,
      },
      {
        createdBy: 'RUN apk add --no-cache --virtual .build-deps-yarn curl gnupg tar',
        size: 5.49 * MB,
      },
      { createdBy: 'COPY docker-entrypoint.sh /usr/local/bin/ # buildkit', size: 20480 },
      { createdBy: 'ENTRYPOINT ["docker-entrypoint.sh"]', size: 0 },
      { createdBy: 'WORKDIR /app', size: 0 },
      { createdBy: 'COPY . . # buildkit', size: 258 * MB },
      { createdBy: 'EXPOSE map[3000/tcp:{}]', size: 0 },
      { createdBy: 'CMD ["node" "server.js"]', size: 0 },
    ],
    architecture: 'arm64',
    os: 'linux',
    entrypoint: ['docker-entrypoint.sh'],
    cmd: ['node', 'server.js'],
    workingDir: '/app',
    user: 'node',
    env: [
      'PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin',
      'NODE_VERSION=22.23.4',
    ],
    exposedPorts: ['3000/tcp'],
  })

export const mockPlanImage = (ref: string): Promise<ImagePlan> =>
  delay({ ref, command: `docker rmi ${ref}`, run: `docker run --rm -it ${ref}` })

export function mockRemoveImage(ref: string): Promise<void> {
  images = images.filter((i) => (i.dangling ? i.id : `${i.repository}:${i.tag}`) !== ref)
  return delay(undefined, 400)
}

let volumes: Volume[] = [
  ['shop_pgdata', 1.4 * 1024 * MB, 1, 'shop'],
  ['shop_redis', 12 * MB, 1, 'shop'],
  ['uploads', 438.6 * MB, 1, ''],
  ['0a1b5a07404e73f21101ddc3cdadd630d36cc80f04883250ec280fa0a0a0a0a0', 227 * 1024, 0, ''],
  ['old_backup', 785 * MB, 0, ''],
].map(([name, size, containers, project]) => ({
  name: name as string,
  driver: 'local',
  mountpoint: `/var/lib/docker/volumes/${name}/_data`,
  created: '2026-10-28T01:37:55Z',
  size: size as number,
  containers: containers as number,
  anonymous: (name as string).length === 64,
  project: (project as string) || undefined,
}))

export const mockVolumes = () => delay(volumes.map((v) => ({ ...v })))

export const mockPlanVolume = (name: string): Promise<VolumePlan> =>
  delay({ name, command: `docker volume rm ${name}` })

export function mockRemoveVolume(name: string): Promise<void> {
  volumes = volumes.filter((v) => v.name !== name)
  return delay(undefined, 300)
}
