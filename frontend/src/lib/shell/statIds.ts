// The readings the menu bar can show (menuStats.ts draws them). Apart from
// it so the settings store can name them without loading the formatters.

/** Every reading the bar can show, in the order it shows them. */
export const STAT_IDS = [
  'cpu',
  'mem',
  'disk',
  'net',
  'load',
  'io',
  'gpu',
  'swap',
  'uptime',
] as const

export type StatId = (typeof STAT_IDS)[number]

/** Shown until the user picks: GPU only appears on a server that has one. */
export const DEFAULT_STATS: readonly StatId[] = ['cpu', 'mem', 'disk', 'net', 'load', 'io', 'gpu']
