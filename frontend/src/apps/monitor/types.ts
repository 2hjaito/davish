import { t } from '../../lib/i18n'
import type { Series } from '../../lib/stores/metrics.svelte'

export type Section =
  'overview' | 'processes' | 'cpu' | 'memory' | 'gpu' | 'disk' | 'network' | 'info'

export type RangeKey = '1m' | '5m' | '15m'

export const RANGES: Record<RangeKey, number> = {
  '1m': 60_000,
  '5m': 5 * 60_000,
  '15m': 15 * 60_000,
}

/** What every chart view draws from. */
export interface ViewProps {
  data: Series
  range: number
  rangeLabel: string
}

export const chartColors = ['var(--chart-1)', 'var(--chart-2)', 'var(--chart-3)'] as const

/** Memory share in percent, 0 when the total is unknown. */
export const memPct = (used: number, total: number) => (total > 0 ? (used / total) * 100 : 0)

export const sectionLabels: Record<Section, string> = {
  overview: t.monitor.overview,
  processes: t.monitor.processes,
  cpu: t.monitor.cpu,
  memory: t.monitor.memory,
  gpu: t.monitor.gpu,
  disk: t.monitor.disk,
  network: t.monitor.network,
  info: t.monitor.info,
}
