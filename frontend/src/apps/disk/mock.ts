// Demo disk for the browser preview: a VPS whose /var is filling up.
import type { Cleanup, Mount, Usage } from '../../lib/api/disk'

const GB = 1 << 30
const MB = 1 << 20
const delay = <T>(v: T, ms = 300) => new Promise<T>((r) => setTimeout(() => r(v), ms))

export function mockMounts(): Promise<Mount[]> {
  return delay([
    { path: '/', total: 40 * GB, used: 35.2 * GB },
    { path: '/boot', total: 1 * GB, used: 0.2 * GB },
  ])
}

const tree: Record<string, [string, number][]> = {
  '/': [
    ['var', 21 * GB],
    ['usr', 6.1 * GB],
    ['home', 4.8 * GB],
    ['opt', 2.2 * GB],
    ['srv', 0.9 * GB],
    ['etc', 12 * MB],
  ],
  '/var': [
    ['lib', 14.5 * GB],
    ['log', 5.3 * GB],
    ['cache', 1.1 * GB],
    ['www', 0.1 * GB],
  ],
  '/var/lib': [
    ['docker', 12.9 * GB],
    ['mysql', 1.4 * GB],
    ['apt', 0.2 * GB],
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
    { id: 'docker', bytes: 6.4 * GB, command: 'docker system prune -f', root: false },
    { id: 'journal', bytes: 3.8 * GB, command: 'journalctl --vacuum-size=100M', root: true },
    {
      id: 'logs',
      bytes: 0.9 * GB,
      count: 214,
      command: `find /var/log -xdev -type f \\( -name '*.gz' -o -name '*.[0-9]' -o -name '*.old' \\) -delete`,
      root: true,
    },
    { id: 'apt', bytes: 0.4 * GB, command: 'apt-get clean', root: true },
  ]
  return delay(all.filter((c) => !cleaned.has(c.id)))
}

export function mockClean(id: string): Promise<void> {
  cleaned = new Set([...cleaned, id])
  return delay(undefined, 800)
}
