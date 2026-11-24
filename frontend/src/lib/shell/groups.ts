// Groups in All servers: a server may name a folder (`group`, one level), and
// the sidebar filters by connection state, group or tag. Pure helpers.

export interface Listed {
  group?: string
  tag?: string
  state: string
}

/** `all`, a connection state, `ungrouped`, `group:<name>` or `tag:<tag>`. */
export type FilterId = string

export const STATE_FILTERS = ['all', 'connected', 'failed', 'disconnected'] as const
export type StateFilter = (typeof STATE_FILTERS)[number]

const key = (s: string | undefined) => (s ?? '').trim().toLowerCase()

/** The state bucket of a server: connecting and reconnecting count as not connected yet. */
export function stateOf(state: string): Exclude<StateFilter, 'all'> {
  return state === 'connected' ? 'connected' : state === 'failed' ? 'failed' : 'disconnected'
}

export interface Named {
  /** The spelling shown: the first one seen. */
  name: string
  count: number
}

function names(list: readonly Listed[], pick: (s: Listed) => string | undefined): Named[] {
  const seen = new Map<string, Named>()
  for (const s of list) {
    const name = pick(s)?.trim()
    if (!name) continue
    const n = seen.get(key(name))
    if (n) n.count++
    else seen.set(key(name), { name, count: 1 })
  }
  return [...seen.values()]
}

/** The groups in use, in the order the servers first mention them. */
export const groupNames = (list: readonly Listed[]): Named[] => names(list, (s) => s.group)

/** The tags in use, in the order the servers first mention them. */
export const tagNames = (list: readonly Listed[]): Named[] => names(list, (s) => s.tag)

/** How many servers sit in no group. */
export const ungroupedCount = (list: readonly Listed[]): number =>
  list.filter((s) => !s.group?.trim()).length

export const groupFilter = (name: string): FilterId => `group:${name}`
export const tagFilter = (name: string): FilterId => `tag:${name}`

/** Whether a server passes a filter; `all` keeps everything. */
export function matches(s: Listed, id: FilterId): boolean {
  if (id === 'all') return true
  if (id === 'ungrouped') return !s.group?.trim()
  if (id.startsWith('group:')) return key(s.group) === key(id.slice(6))
  if (id.startsWith('tag:')) return key(s.tag) === key(id.slice(4))
  return stateOf(s.state) === id
}

/**
 * The filter to apply: the chosen one while it still means something, `all`
 * once its group or tag is gone, so a filter never leaves an empty list behind.
 */
export function activeFilter(list: readonly Listed[], chosen: FilterId): FilterId {
  if (chosen.startsWith('group:') || chosen.startsWith('tag:') || chosen === 'ungrouped')
    return list.some((s) => matches(s, chosen)) ? chosen : 'all'
  return (STATE_FILTERS as readonly string[]).includes(chosen) ? chosen : 'all'
}

export interface Section<T> {
  /** The folder's name; empty for servers in none. */
  name: string
  servers: T[]
}

/**
 * The servers as folders, then the ungrouped ones. Folders follow `order`
 * (names, usually the sidebar's) so they do not jump around when the servers
 * inside are re-sorted; any not in it come after, as they first appear. With
 * no group at all there is one nameless section, so a list without groups
 * looks exactly as before.
 */
export function sections<T extends Listed>(
  list: readonly T[],
  order: readonly string[] = [],
): Section<T>[] {
  const named = new Map<string, Section<T>>()
  const loose: T[] = []
  for (const s of list) {
    const g = s.group?.trim()
    if (!g) loose.push(s)
    else {
      const sec = named.get(key(g))
      if (sec) sec.servers.push(s)
      else named.set(key(g), { name: g, servers: [s] })
    }
  }
  if (named.size === 0) return [{ name: '', servers: loose }]
  const rank = (name: string) => {
    const i = order.findIndex((o) => key(o) === key(name))
    return i < 0 ? order.length : i
  }
  // Array.sort is stable, so groups outside `order` keep their order of appearance.
  const out = [...named.values()].sort((a, b) => rank(a.name) - rank(b.name))
  if (loose.length) out.push({ name: '', servers: loose })
  return out
}

/**
 * The servers in the order they are shown and numbered: folder by folder,
 * the ungrouped last. `⌘1…⌘9` follow this order, so group A reads 1, 2, 3 and
 * group B 4, 5, 6. With no group it is the list itself, in its own order.
 */
export function displayOrder<T extends Listed>(list: readonly T[]): T[] {
  return sections(list).flatMap((sec) => sec.servers)
}
