// Demo disk for the browser preview: a VPS whose /var is filling up.
import type { Cleanup, Mount, Overview, TruncatePlan, Usage } from '../../lib/api/disk'
import type { RemovePlan } from '../../lib/api/files'

const GB = 1 << 30
const MB = 1 << 20
const delay = <T>(v: T, ms = 300) => new Promise<T>((r) => setTimeout(() => r(v), ms))

export function mockMounts(): Promise<Mount[]> {
  return delay([
    { path: '/', total: 40 * GB, used: 36.4 * GB },
    { path: '/boot', total: 1 * GB, used: 0.2 * GB },
  ])
}

const tree: Record<string, [string, number][]> = {
  '/': [
    ['var', 18 * GB],
    ['usr', 6.1 * GB],
    ['home', 4.8 * GB],
    ['opt', 2.2 * GB],
    ['srv', 0.9 * GB],
    ['etc', 12 * MB],
  ],
  '/var': [
    ['lib', 11.5 * GB],
    ['log', 5.3 * GB],
    ['cache', 1.1 * GB],
    ['www', 0.1 * GB],
  ],
  '/var/lib': [
    ['docker', 9.8 * GB],
    ['mysql', 1.4 * GB],
    ['apt', 0.2 * GB],
  ],
  '/var/lib/docker': [
    ['containers', 6.8 * GB],
    ['overlay2', 2.6 * GB],
    ['volumes', 0.4 * GB],
  ],
  '/home': [
    ['deploy', 4.7 * GB],
    ['ubuntu', 0.1 * GB],
  ],
  '/home/deploy': [
    ['backups', 3.1 * GB],
    ['apps', 1.4 * GB],
    ['.cache', 0.2 * GB],
  ],
  '/opt': [['shop', 2.2 * GB]],
  '/opt/shop': [
    ['node_modules', 1.6 * GB],
    ['releases', 0.6 * GB],
  ],
  '/var/log': [
    ['journal', 3.9 * GB],
    ['nginx', 1.1 * GB],
    ['apt', 18 * MB],
  ],
}

const canceled = new Set<string>()

export async function mockUsage(dir: string, token: string): Promise<Usage> {
  await delay(null, 900)
  if (canceled.delete(token)) throw new Error('canceled')
  const kids = tree[dir] ?? []
  const base = dir === '/' ? '' : dir
  return {
    dir,
    size: kids.reduce((s, [, n]) => s + n, 0),
    entries: kids.map(([name, size]) => ({ path: `${base}/${name}`, size })),
    partial: false,
  }
}

export function mockCancel(token: string): Promise<void> {
  canceled.add(token)
  return delay(undefined, 0)
}

let cleaned = new Set<string>()

export function mockCleanups(): Promise<Cleanup[]> {
  const all: Cleanup[] = [
    { id: 'docker-images', bytes: 6.4 * GB, command: 'docker image prune -a -f', root: false },
    {
      id: 'docker-logs',
      bytes: 4.2 * GB,
      count: 3,
      command: `find /var/lib/docker/containers -name '*-json.log' -size +10M -exec truncate -s 0 {} +`,
      root: true,
      sudoCommand: `sudo find /var/lib/docker/containers -name '*-json.log' -size +10M -exec truncate -s 0 {} +`,
    },
    {
      id: 'journal',
      bytes: 3.8 * GB,
      command: 'journalctl --vacuum-size=100M',
      root: true,
      sudoCommand: 'sudo journalctl --vacuum-size=100M',
    },
    { id: 'docker-cache', bytes: 1.7 * GB, command: 'docker builder prune -a -f', root: false },
    {
      id: 'logs',
      bytes: 0.9 * GB,
      count: 214,
      command: `find /var/log -xdev -type f \\( -name '*.gz' -o -name '*.[0-9]' -o -name '*.old' \\) -delete`,
      root: true,
      sudoCommand: `sudo find /var/log -xdev -type f \\( -name '*.gz' -o -name '*.[0-9]' -o -name '*.old' \\) -delete`,
    },
    {
      id: 'docker-volumes',
      bytes: 0.8 * GB,
      command: 'docker volume prune -a -f',
      root: false,
      danger: true,
    },
    {
      id: 'apt',
      bytes: 0.4 * GB,
      command: 'apt-get clean',
      root: true,
      sudoCommand: 'sudo apt-get clean',
    },
    {
      id: 'home-cache',
      bytes: 0.3 * GB,
      command: 'find "$HOME/.cache" -mindepth 1 -delete',
      root: false,
    },
  ]
  return delay(all.filter((c) => !cleaned.has(c.id)))
}

export function mockClean(id: string): Promise<void> {
  cleaned = new Set([...cleaned, id])
  return delay(undefined, 800)
}

const removed = new Set<string>()

/** The tree's folders down to three levels, as the overview lists them. */
function mockDirs(root: string, depth: number): { path: string; size: number }[] {
  if (depth === 0) return []
  const base = root === '/' ? '' : root
  return (tree[root] ?? [])
    .map(([name, size]) => ({ path: `${base}/${name}`, size }))
    .filter((e) => !removed.has(e.path))
    .flatMap((e) => [e, ...mockDirs(e.path, depth - 1)])
}

export async function mockOverview(mount: string, token: string): Promise<Overview> {
  await delay(null, 1400)
  if (canceled.delete(token)) throw new Error('canceled')
  const m = (await mockMounts()).find((x) => x.path === mount) ?? {
    path: mount,
    total: GB,
    used: 0,
  }
  const dirs = mockDirs(mount, 3).sort((a, b) => b.size - a.size)
  const measured = (tree[mount] ?? []).reduce((s, [, n]) => s + n, 0)
  const files = [
    { path: '/var/lib/docker/containers/3f9c2a/3f9c2a-json.log', size: 6.1 * GB },
    { path: '/home/deploy/backups/shop-2026-10-01.sql.gz', size: 1.6 * GB },
    { path: '/home/deploy/backups/shop-2026-09-01.sql.gz', size: 1.4 * GB },
    { path: '/var/log/nginx/access.log.1', size: 0.6 * GB },
    { path: '/var/lib/mysql/ibdata1', size: 0.4 * GB },
  ].filter((f) => mount === '/' && !removed.has(f.path))
  const deleted =
    mount === '/'
      ? [{ path: '/var/log/shop/worker.log', size: 1.8 * GB, pid: 2214, command: 'node' }]
      : []
  const held = deleted.reduce((s, d) => s + d.size, 0)
  return {
    mount: m,
    measured,
    dirs,
    files,
    unreadable: mount === '/' ? ['/root', '/var/lib/docker/volumes'] : [],
    deleted,
    unmeasured: Math.max(0, m.used - measured - held),
    root: false,
    partial: false,
    sudoMeasure: `sudo du -xh -d 2 -- ${mount} 2>/dev/null | sort -h | tail -n 25`,
    askSudo: false,
  }
}

// Like internal/disk.Protected, for the folders the demo has.
const PROTECTED = ['/usr', '/etc', '/var/lib/docker', '/var/lib/mysql', '/var/lib/apt']

export function mockPlanRemove(path: string): Promise<RemovePlan> {
  if (
    PROTECTED.some((p) => path === p || path.startsWith(p + '/')) ||
    ['/', '/var', '/var/lib', '/var/log', '/home', '/opt', '/srv', '/var/cache'].includes(path) ||
    /^\/home\/[^/]+$/.test(path)
  )
    return Promise.reject({ code: 'INVALID_INPUT' })
  const name = path.slice(path.lastIndexOf('/') + 1)
  const dir = !name.includes('.')
  const command = dir ? `rm -r -- ${path}` : `rm -- ${path}`
  return delay({
    name,
    path,
    kind: dir ? 'dir' : 'file',
    dir,
    size: 0,
    mode: dir ? 'drwxr-xr-x' : '-rw-r--r--',
    perm: 0o755,
    modTime: 0,
    uid: 1000,
    gid: 1000,
    nonEmpty: dir,
    command,
    sudo: `sudo ${command}`,
  })
}

export function mockRemove(path: string): Promise<void> {
  removed.add(path)
  return delay(undefined, 500)
}

export function mockPlanTruncate(path: string): Promise<TruncatePlan> {
  return delay({ command: `truncate -s 0 -- ${path}`, sudo: `sudo truncate -s 0 -- ${path}` })
}
