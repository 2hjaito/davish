// What the Disk widget makes of the mounts the collector already reads
// (ADR 0026): no command of its own. The level follows the threshold the
// person set for the disk alert, so the widget turns red when the alert would.
import type { Mount } from '../api/events'

export type DiskLevel = '' | 'warn' | 'danger'

/** How close to the alert threshold a mount turns amber, in percentage points. */
export const WARN_BELOW = 10

/** danger from the threshold, warn within WARN_BELOW of it. */
export function levelOf(pct: number, alertPct: number): DiskLevel {
  if (!Number.isFinite(pct)) return ''
  if (pct >= alertPct) return 'danger'
  if (pct >= alertPct - WARN_BELOW) return 'warn'
  return ''
}

export interface DiskRow {
  path: string
  pct: number
  used: number
  size: number
  level: DiskLevel
}

export interface DiskSummary {
  /** The fullest mount, or null when none was read. */
  fullest: DiskRow | null
  /** The worst level of any mount. */
  level: DiskLevel
  /** Every mount, the fullest first. */
  rows: DiskRow[]
}

const RANK: Record<DiskLevel, number> = { '': 0, warn: 1, danger: 2 }

export function summarizeMounts(mounts: readonly Mount[], alertPct: number): DiskSummary {
  const rows = mounts
    .filter((m) => m.size > 0)
    .map<DiskRow>((m) => ({
      path: m.path,
      pct: m.pct,
      used: m.used,
      size: m.size,
      level: levelOf(m.pct, alertPct),
    }))
    .sort((a, b) => b.pct - a.pct || a.path.localeCompare(b.path))
  const level = rows.reduce<DiskLevel>((w, r) => (RANK[r.level] > RANK[w] ? r.level : w), '')
  return { fullest: rows[0] ?? null, level, rows }
}
