// Demo data for the plain browser preview (vite dev), where there is no Go side.
import type { Process, ProcessList, ProcessSort, SystemInfo } from '../../lib/api/monitor'

const MB = 1024 ** 2

const base: Omit<Process, 'cpu'>[] = [
  {
    pid: 1,
    ppid: 0,
    user: 'root',
    name: 'systemd',
    command: '/sbin/init',
    state: 'S',
    mem: 12 * MB,
    memPct: 0.1,
    threads: 1,
    age: 390_000,
  },
  {
    pid: 812,
    ppid: 1,
    user: 'root',
    name: 'dockerd',
    command: '/usr/bin/dockerd -H fd://',
    state: 'S',
    mem: 96 * MB,
    memPct: 0.6,
    threads: 18,
    age: 389_000,
  },
  {
    pid: 1022,
    ppid: 1,
    user: 'postgres',
    name: 'postgres',
    command: '/usr/lib/postgresql/16/bin/postgres -D /var/lib/postgresql/16/main',
    state: 'S',
    mem: 420 * MB,
    memPct: 2.6,
    threads: 1,
    age: 388_000,
  },
  {
    pid: 1310,
    ppid: 1,
    user: 'www-data',
    name: 'nginx',
    command: 'nginx: worker process',
    state: 'S',
    mem: 18 * MB,
    memPct: 0.1,
    threads: 1,
    age: 120_000,
  },
  {
    pid: 2201,
    ppid: 1,
    user: 'app',
    name: 'node',
    command: 'node /srv/app/dist/server.js --port 3000',
    state: 'R',
    mem: 610 * MB,
    memPct: 3.8,
    threads: 11,
    age: 86_000,
  },
  {
    pid: 2290,
    ppid: 2201,
    user: 'app',
    name: 'node',
    command: 'node /srv/app/dist/worker.js',
    state: 'S',
    mem: 280 * MB,
    memPct: 1.7,
    threads: 7,
    age: 86_000,
  },
  {
    pid: 3105,
    ppid: 1,
    user: 'redis',
    name: 'redis-server',
    command: '/usr/bin/redis-server 127.0.0.1:6379',
    state: 'S',
    mem: 54 * MB,
    memPct: 0.3,
    threads: 5,
    age: 380_000,
  },
  {
    pid: 4410,
    ppid: 1,
    user: 'root',
    name: 'sshd',
    command: 'sshd: /usr/sbin/sshd -D [listener]',
    state: 'S',
    mem: 8 * MB,
    memPct: 0,
    threads: 1,
    age: 390_000,
  },
  {
    pid: 5120,
    ppid: 4410,
    user: 'dangth',
    name: 'bash',
    command: '-bash',
    state: 'S',
    mem: 5 * MB,
    memPct: 0,
    threads: 1,
    age: 1_800,
  },
  {
    pid: 6001,
    ppid: 1,
    user: 'root',
    name: 'kworker/0:1',
    command: '[kworker/0:1]',
    state: 'I',
    mem: 0,
    memPct: 0,
    threads: 1,
    age: 9_000,
  },
]

const load = [0.3, 1.2, 4.5, 0.8, 38, 12, 2.1, 0, 0, 0.2]

export function mockProcesses(sortBy: ProcessSort): ProcessList {
  const procs = base.map((p, i) => ({
    ...p,
    cpu: Math.max(
      0,
      Math.round(((load[i] ?? 0) + (Math.random() * 2 - 1) * (load[i] ?? 0) * 0.3) * 10) / 10,
    ),
  }))
  procs.sort((a, b) => (sortBy === 'mem' ? b.mem - a.mem : b.cpu - a.cpu) || a.pid - b.pid)
  return { total: 214, procs }
}

export const mockInfo: SystemInfo = {
  hostname: 'dangth-vps',
  os: 'Ubuntu 24.04.1 LTS',
  kernel: '6.8.0-45-generic',
  arch: 'x86_64',
  cpuModel: 'AMD EPYC 7B13',
  virt: 'kvm',
}
