// Demo ticks for the plain browser preview (vite dev), where there is no Go
// side. The shape matches sysmon:tick so every Monitor view has something to show.
import type { Metrics } from './metrics.svelte'

const GB = 1024 ** 3
const MB = 1024 ** 2

const jitter = (v: number, d: number, lo: number, hi: number) =>
  Math.min(hi, Math.max(lo, v + (Math.random() * 2 - 1) * d))

const round1 = (v: number) => Math.round(v * 10) / 10

export function firstMockTick(): Metrics {
  return {
    cpu: 23,
    memUsed: 5.1 * GB,
    memTotal: 16 * GB,
    diskPct: 62,
    rxBps: 1.2 * MB,
    txBps: 0.2 * MB,
    load1: 0.42,
    uptime: 4 * 86400 + 13 * 3600,
    load5: 0.38,
    load15: 0.31,
    procs: 214,
    cpuSplit: { user: 15, system: 6, iowait: 1.5, steal: 0.5 },
    cores: [31, 18, 27, 12, 40, 9, 22, 25],
    memCache: 6.4 * GB,
    memFree: 4.5 * GB,
    swapTotal: 2 * GB,
    swapUsed: 0.3 * GB,
    readBps: 120 * 1024,
    writeBps: 860 * 1024,
    disks: [{ name: 'vda', readBps: 120 * 1024, writeBps: 860 * 1024 }],
    mounts: [
      { device: '/dev/vda1', path: '/', size: 80 * GB, used: 47 * GB, avail: 29 * GB, pct: 62 },
      {
        device: '/dev/vda15',
        path: '/boot/efi',
        size: 105 * MB,
        used: 6 * MB,
        avail: 99 * MB,
        pct: 6,
      },
      {
        device: '/dev/vdb',
        path: '/data',
        size: 500 * GB,
        used: 431 * GB,
        avail: 44 * GB,
        pct: 91,
      },
    ],
    ifaces: [
      { name: 'eth0', rxBps: 1.2 * MB, txBps: 0.2 * MB, rxTotal: 182 * GB, txTotal: 41 * GB },
      { name: 'wg0', rxBps: 0, txBps: 0, rxTotal: 3 * GB, txTotal: 2 * GB },
    ],
    gpus: [],
  }
}

export function nextMockTick(c: Metrics): Metrics {
  const cores = c.cores.map((v) => Math.round(jitter(v, 10, 1, 99)))
  const cpu = round1(cores.reduce((a, b) => a + b, 0) / cores.length)
  const rxBps = jitter(c.rxBps, 300 * 1024, 20 * 1024, 4 * MB)
  const txBps = jitter(c.txBps, 80 * 1024, 5 * 1024, MB)
  const readBps = jitter(c.readBps, 200 * 1024, 0, 8 * MB)
  const writeBps = jitter(c.writeBps, 400 * 1024, 0, 12 * MB)
  return {
    ...c,
    cpu,
    cores,
    cpuSplit: {
      user: round1(cpu * 0.68),
      system: round1(cpu * 0.26),
      iowait: round1(cpu * 0.04),
      steal: round1(cpu * 0.02),
    },
    memUsed: jitter(c.memUsed, 0.15 * GB, 3 * GB, 9 * GB),
    rxBps,
    txBps,
    readBps,
    writeBps,
    disks: [{ name: 'vda', readBps, writeBps }],
    ifaces: c.ifaces.map((n, i) =>
      i === 0
        ? { ...n, rxBps, txBps, rxTotal: n.rxTotal + rxBps * 2, txTotal: n.txTotal + txBps * 2 }
        : n,
    ),
    uptime: c.uptime + 2,
  }
}
