// What each reading in the menu bar shows: a ring with a number in it (or
// an icon and a short value) in the bar, a larger value, a line of detail
// and a 15 minute chart on hover.
// Pure, so it can be tested without the bar.
import {
  formatBytes,
  formatDuration,
  formatGB,
  formatPct,
  formatRate,
  formatRateShort,
} from '../format'
import { t } from '../i18n'
import type { Sample } from '../stores/metrics.svelte'
import { type ChartSeries, level, type Level } from '../ui/charts/scale'
import type { IconName } from '../ui/icons'
import { STAT_IDS, type StatId } from './statIds'

export { DEFAULT_STATS, STAT_IDS, type StatId } from './statIds'

const COLORS = ['var(--chart-1)', 'var(--chart-2)'] as const

export interface StatChart {
  series: ChartSeries<Sample>[]
  format: (v: number) => string
  /** Fixed top of the axis; otherwise fitted to the data. */
  max?: number
  binary?: boolean
  floor?: number
}

export interface StatView {
  id: StatId
  label: string
  /** How full the ring is, 0 to 1; null draws icon instead. */
  ring: number | null
  icon: IconName
  /** The name beside the ring in the bar; empty where the icon says it. */
  short: string
  /** The number inside the ring; empty where the icon sits there instead. */
  inner: string
  /** The short value; beside the icon in the bar, and read out for the ring. */
  text: string
  level: Level
  /** The large value on hover. */
  hero: string
  /** A line of detail beside or under hero. */
  sub: string
  chart?: StatChart
}

export const loadFormat = (v: number) => v.toFixed(v < 10 ? 2 : 1)

const clamp01 = (v: number) => Math.min(1, Math.max(0, Number.isFinite(v) ? v : 0))

/** A share as a whole number that fits in the ring, 0 to 100. */
export const ringPct = (pct: number) => String(Math.round(clamp01(pct / 100) * 100))

/** Load in at most three characters, so it fits in the ring. */
export const ringLoad = (v: number) => (v < 9.95 ? v.toFixed(1) : String(Math.round(v)))

/** The view of one reading; null when the server has nothing to show for it. */
export function statView(id: StatId, s: Sample): StatView | null {
  const m = t.menubar.stats
  switch (id) {
    case 'cpu':
      return {
        id,
        label: t.menubar.cpu,
        ring: clamp01(s.cpu / 100),
        icon: 'cpu',
        short: m.short.cpu,
        inner: ringPct(s.cpu),
        text: formatPct(s.cpu),
        level: level(s.cpu, 85, 95),
        hero: formatPct(s.cpu),
        sub: m.cpuSub(s.cores.length, loadFormat(s.load1)),
        chart: {
          series: [{ label: t.menubar.cpu, color: COLORS[0], value: (x) => x.cpu }],
          format: formatPct,
          max: 100,
        },
      }
    case 'mem': {
      const pct = s.memTotal > 0 ? (s.memUsed / s.memTotal) * 100 : 0
      return {
        id,
        label: t.menubar.ram,
        ring: clamp01(pct / 100),
        icon: 'memory',
        short: m.short.mem,
        inner: ringPct(pct),
        text: `${formatGB(s.memUsed)} GB`,
        level: level(pct, 90, 95),
        hero: formatBytes(s.memUsed),
        sub: m.memSub(formatBytes(s.memTotal), formatPct(pct)),
        chart: {
          series: [{ label: t.menubar.ram, color: COLORS[0], value: (x) => x.memUsed }],
          format: formatBytes,
          max: s.memTotal || undefined,
          binary: true,
        },
      }
    }
    case 'disk':
      return {
        id,
        label: t.menubar.disk,
        ring: clamp01(s.diskPct / 100),
        icon: 'disk',
        short: m.short.disk,
        inner: ringPct(s.diskPct),
        text: formatPct(s.diskPct),
        level: level(s.diskPct, 85, 95),
        hero: formatPct(s.diskPct),
        sub: t.monitor.rootDisk,
        chart: {
          series: [{ label: t.menubar.disk, color: COLORS[0], value: (x) => x.diskPct }],
          format: formatPct,
          max: 100,
        },
      }
    case 'net':
      return {
        id,
        label: t.menubar.network,
        ring: null,
        icon: 'network',
        short: '',
        inner: '',
        text: formatRateShort(s.rxBps),
        level: '',
        hero: formatRate(s.rxBps),
        sub: m.netSub(formatRate(s.txBps)),
        chart: {
          series: [
            { label: t.monitor.rx, color: COLORS[0], value: (x) => x.rxBps },
            { label: t.monitor.tx, color: COLORS[1], value: (x) => x.txBps },
          ],
          format: formatRate,
          binary: true,
          floor: 1024,
        },
      }
    case 'load': {
      const cores = Math.max(1, s.cores.length)
      const ratio = s.load1 / cores
      return {
        id,
        label: m.names.load,
        ring: clamp01(ratio),
        icon: 'monitor',
        short: m.short.load,
        inner: ringLoad(s.load1),
        text: loadFormat(s.load1),
        // Load equal to the core count means every core is busy.
        level: level(ratio * 100, 100, 150),
        hero: loadFormat(s.load1),
        sub: m.loadSub(loadFormat(s.load5), loadFormat(s.load15), cores),
        chart: {
          series: [{ label: m.names.load, color: COLORS[0], value: (x) => x.load1 }],
          format: loadFormat,
          floor: cores,
        },
      }
    }
    case 'io':
      return {
        id,
        label: m.names.io,
        ring: null,
        icon: 'disk',
        short: m.short.io,
        inner: '',
        text: formatRateShort(s.readBps + s.writeBps),
        level: '',
        hero: formatRate(s.readBps + s.writeBps),
        sub: m.ioSub(formatRate(s.readBps), formatRate(s.writeBps)),
        chart: {
          series: [
            { label: t.monitor.read, color: COLORS[0], value: (x) => x.readBps },
            { label: t.monitor.write, color: COLORS[1], value: (x) => x.writeBps },
          ],
          format: formatRate,
          binary: true,
          floor: 1024,
        },
      }
    case 'gpu': {
      const g = s.gpus[0]
      if (!g) return null
      return {
        id,
        label: m.names.gpu,
        ring: clamp01(g.util / 100),
        icon: 'gpu',
        short: m.short.gpu,
        inner: ringPct(g.util),
        text: formatPct(g.util),
        level: level(g.util, 90, 98),
        hero: formatPct(g.util),
        sub: m.gpuSub(g.name, `${formatBytes(g.memUsed)} / ${formatBytes(g.memTotal)}`, g.temp),
        chart: {
          series: [{ label: m.names.gpu, color: COLORS[0], value: (x) => x.gpus[0]?.util ?? 0 }],
          format: formatPct,
          max: 100,
        },
      }
    }
    case 'swap': {
      if (s.swapTotal <= 0) return null
      const pct = (s.swapUsed / s.swapTotal) * 100
      return {
        id,
        label: m.names.swap,
        ring: clamp01(pct / 100),
        icon: 'memory',
        short: m.short.swap,
        inner: ringPct(pct),
        text: formatPct(pct),
        level: level(pct, 50, 80),
        hero: formatBytes(s.swapUsed),
        sub: m.swapSub(formatBytes(s.swapTotal)),
        chart: {
          series: [{ label: m.names.swap, color: COLORS[0], value: (x) => x.swapUsed }],
          format: formatBytes,
          max: s.swapTotal,
          binary: true,
        },
      }
    }
    case 'uptime':
      return {
        id,
        label: m.names.uptime,
        ring: null,
        icon: 'clock',
        short: '',
        inner: '',
        text: formatDuration(s.uptime, t.units),
        level: '',
        hero: formatDuration(s.uptime, t.units),
        sub: m.uptimeSub(s.procs),
      }
  }
}

/** The views to draw, in bar order, skipping readings the server lacks. */
export function statViews(ids: readonly StatId[], s: Sample): StatView[] {
  return STAT_IDS.filter((id) => ids.includes(id))
    .map((id) => statView(id, s))
    .filter((v): v is StatView => v !== null)
}
