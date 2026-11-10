// Pure helpers of the All servers window.
import type { Level, Notification } from '../../lib/stores/notifications.svelte'

export interface Alerts {
  count: number
  /** The most serious level among them. */
  level: Level | null
  /** The newest one's title. */
  latest: string
}

const rank: Record<Level, number> = { info: 0, warn: 1, error: 2 }

/** What the notifications say about one server. */
export function alertsFor(list: readonly Notification[], serverID: string): Alerts {
  const mine = list.filter((n) => n.serverID === serverID)
  let level: Level | null = null
  let latest: Notification | undefined
  for (const n of mine) {
    if (!level || rank[n.level] > rank[level]) level = n.level
    if (!latest || n.at > latest.at) latest = n
  }
  return { count: mine.length, level, latest: latest?.title ?? '' }
}

/** RAM in use as a percentage, 0 without a total. */
export function memPct(used: number, total: number): number {
  return total > 0 ? (used / total) * 100 : 0
}
