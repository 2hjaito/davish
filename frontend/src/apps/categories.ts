// The groups Launchpad sorts apps into.
import type { AppId } from './types'

export type Category = 'files' | 'monitor' | 'web' | 'security' | 'dev' | 'system'

export const CATEGORIES: readonly Category[] = [
  'files',
  'monitor',
  'web',
  'security',
  'dev',
  'system',
]

/** Every app's group; a new app does not compile until it has one. */
export const CATEGORY: Record<AppId, Category> = {
  terminal: 'files',
  files: 'files',
  editor: 'files',
  transfers: 'files',
  monitor: 'monitor',
  services: 'monitor',
  logs: 'monitor',
  disk: 'monitor',
  audit: 'monitor',
  servers: 'monitor',
  docker: 'web',
  nginx: 'web',
  browser: 'web',
  ports: 'web',
  security: 'security',
  keys: 'security',
  git: 'dev',
  deploy: 'dev',
  database: 'dev',
  packages: 'system',
  cron: 'system',
  settings: 'system',
}

/** The apps of each group that has any, groups and apps in their order. */
export function grouped<T extends { id: AppId }>(
  list: readonly T[],
): { category: Category; apps: T[] }[] {
  return CATEGORIES.map((category) => ({
    category,
    apps: list.filter((a) => CATEGORY[a.id] === category),
  })).filter((g) => g.apps.length > 0)
}

/**
 * Where an arrow key goes in grids of cols columns stacked one after the
 * other, each sizes[k] tiles long; index counts across all of them. Left
 * and right step through the tiles; up and down keep the column, moving
 * into the next or previous grid, onto its last tile when the row there
 * is shorter. Stays put at the edges.
 */
export function gridMove(
  sizes: readonly number[],
  index: number,
  key: 'ArrowLeft' | 'ArrowRight' | 'ArrowUp' | 'ArrowDown',
  cols: number,
): number {
  const total = sizes.reduce((a, b) => a + b, 0)
  if (key === 'ArrowLeft') return Math.max(0, index - 1)
  if (key === 'ArrowRight') return Math.min(total - 1, index + 1)
  // Rows as [first index, length], top to bottom.
  const rows: [number, number][] = []
  let start = 0
  for (const n of sizes) {
    for (let r = 0; r < n; r += cols) rows.push([start + r, Math.min(cols, n - r)])
    start += n
  }
  const row = rows.findIndex(([s, n]) => index >= s && index < s + n)
  const from = rows[row]
  const to = rows[row + (key === 'ArrowDown' ? 1 : -1)]
  if (!from || !to) return index
  return to[0] + Math.min(index - from[0], to[1] - 1)
}
