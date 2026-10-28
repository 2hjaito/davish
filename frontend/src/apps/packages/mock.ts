// Demo data for the plain browser preview (vite dev), where there is no Go side.
import type { Inventory, Manager, NpmGlobal, PackageOp, ServerApp } from '../../lib/api/packages'

const delay = <T>(v: T, ms = 200) => new Promise<T>((r) => setTimeout(() => r(v), ms))

const installed: [string, string, number, string][] = [
  ['curl', '8.5.0-2ubuntu10.4', 520, 'command line tool for transferring data with URL syntax'],
  [
    'docker-ce',
    '5:27.3.1-1~ubuntu.24.04~noble',
    98_000,
    'Docker: the open-source application container engine',
  ],
  ['git', '1:2.43.0-1ubuntu7.1', 23_000, 'fast, scalable, distributed revision control system'],
  ['libssl3t64', '3.0.13-0ubuntu3.4', 7_000, 'Secure Sockets Layer toolkit - shared libraries'],
  ['mariadb-server', '1:10.11.8-0ubuntu0.24.04.1', 64_000, 'MariaDB database server'],
  ['nginx', '1.24.0-2ubuntu7.1', 1_350, 'small, powerful, scalable web/proxy server'],
  ['nodejs', '20.18.0-1nodesource1', 190_000, 'Node.js event-based server-side javascript engine'],
  [
    'openssh-server',
    '1:9.6p1-3ubuntu13.5',
    2_100,
    'secure shell (SSH) server, for secure access from remote machines',
  ],
  ['openssl', '3.0.13-0ubuntu3.4', 2_300, 'Secure Sockets Layer toolkit - cryptographic utility'],
  [
    'php8.3-fpm',
    '8.3.6-0ubuntu0.24.04.2',
    15_000,
    'server-side, HTML-embedded scripting language (FPM-CGI binary)',
  ],
  [
    'python3',
    '3.12.3-0ubuntu2',
    90,
    'interactive high-level object-oriented language (default python3 version)',
  ],
  ['redis-server', '5:7.0.15-1build2', 230, 'Persistent key-value database with network interface'],
]

export const mockInventory = (): Promise<Inventory> =>
  delay({
    manager: 'apt' as Manager,
    installed: installed.map(([name, version, kb, summary]) => ({
      name,
      version,
      size: kb * 1024,
      summary,
    })),
    updates: [
      {
        name: 'libssl3t64',
        current: '3.0.13-0ubuntu3.4',
        available: '3.0.13-0ubuntu3.5',
        security: true,
      },
      {
        name: 'openssl',
        current: '3.0.13-0ubuntu3.4',
        available: '3.0.13-0ubuntu3.5',
        security: true,
      },
      {
        name: 'nginx',
        current: '1.24.0-2ubuntu7.1',
        available: '1.24.0-2ubuntu7.3',
        security: false,
      },
      {
        name: 'curl',
        current: '8.5.0-2ubuntu10.4',
        available: '8.5.0-2ubuntu10.5',
        security: false,
      },
    ],
    listsAge: 9 * 86400,
  })

export const mockApps = (): Promise<ServerApp[]> =>
  delay([
    {
      id: 'nginx',
      label: 'nginx',
      kind: 'server',
      version: '1.24.0',
      path: '/usr/sbin/nginx',
      configs: ['/etc/nginx/nginx.conf'],
      units: ['nginx.service'],
      versions: [],
    },
    {
      id: 'mariadb',
      label: 'MariaDB',
      kind: 'server',
      version: '10.11.8-MariaDB-0ubuntu0.24.04.1',
      path: '/usr/sbin/mariadbd',
      configs: ['/etc/mysql/my.cnf'],
      units: ['mariadb.service'],
      versions: [],
    },
    {
      id: 'redis',
      label: 'Redis',
      kind: 'server',
      version: '7.0.15',
      path: '/usr/bin/redis-server',
      configs: ['/etc/redis/redis.conf'],
      units: ['redis-server.service', 'redis.service'],
      versions: [],
    },
    {
      id: 'docker',
      label: 'Docker',
      kind: 'server',
      version: '27.3.1',
      path: '/usr/bin/docker',
      configs: [],
      units: ['docker.service'],
      versions: [],
    },
    {
      id: 'node',
      label: 'Node.js',
      kind: 'runtime',
      version: '20.18.0',
      path: '/home/dangth/.nvm/versions/node/v20.18.0/bin/node',
      configs: [],
      units: [],
      versions: ['18.20.4', '20.18.0'],
    },
    {
      id: 'npm',
      label: 'npm',
      kind: 'runtime',
      version: '10.8.2',
      path: '/home/dangth/.nvm/versions/node/v20.18.0/bin/npm',
      configs: [],
      units: [],
      versions: [],
    },
    {
      id: 'php',
      label: 'PHP',
      kind: 'runtime',
      version: '8.3.6',
      path: '/usr/bin/php',
      configs: [],
      units: ['php8.3-fpm.service'],
      versions: ['php-fpm8.3'],
    },
    {
      id: 'python',
      label: 'Python',
      kind: 'runtime',
      version: '3.12.3',
      path: '/usr/bin/python3',
      configs: [],
      units: [],
      versions: [],
    },
    {
      id: 'git',
      label: 'Git',
      kind: 'tool',
      version: '2.43.0',
      path: '/usr/bin/git',
      configs: [],
      units: [],
      versions: [],
    },
  ])

export const mockNpmGlobals = (): Promise<NpmGlobal[]> =>
  delay([
    { name: 'corepack', version: '0.29.3' },
    { name: 'pm2', version: '5.4.2' },
  ])

/** The same commands Go plans, for the apt demo server. */
export function mockPlan(_m: Manager, op: PackageOp, names: string[]): Promise<string> {
  const n = names.join(' ')
  const cmd: Record<PackageOp, string> = {
    install: `sudo apt-get install ${n}`,
    remove: `sudo apt-get remove ${n}`,
    upgrade: `sudo apt-get install --only-upgrade ${n}`,
    'upgrade-security': `sudo apt-get install --only-upgrade ${n}`,
    'upgrade-all': 'sudo apt-get update && sudo apt-get upgrade',
    refresh: 'sudo apt-get update',
  }
  return delay(cmd[op], 50)
}
