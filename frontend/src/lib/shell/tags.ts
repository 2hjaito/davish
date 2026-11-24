// Server tags: a short label per server (local, staging, prod…) so the
// wrong environment is harder to open by mistake. Pure helpers; the tint of
// a tag is the server's own `color`.

export interface Tagged {
  tag?: string
  color?: string
}

/** One-click tags in the server dialog, colored from the dialog's palette. */
export const TAG_PRESETS: readonly { tag: string; color: string }[] = [
  { tag: 'local', color: '#16a067' },
  { tag: 'staging', color: '#e08a12' },
  { tag: 'prod', color: '#d6455a' },
]

/** Longest tag the backend accepts (internal/store maxTagLen). */
export const MAX_TAG_LENGTH = 24

const key = (tag: string) => tag.trim().toLowerCase()

export interface TagGroup {
  /** The spelling shown: the first one seen. */
  tag: string
  color?: string
  count: number
}

/**
 * The distinct tags in use, with how many servers carry each. "Prod" and
 * "prod" are one tag. The presets come first, in their usual order, then the
 * rest alphabetically.
 */
export function tagGroups(list: readonly Tagged[]): TagGroup[] {
  const groups = new Map<string, TagGroup>()
  for (const s of list) {
    const tag = s.tag?.trim()
    if (!tag) continue
    const g = groups.get(key(tag))
    if (g) {
      g.count++
      g.color ??= s.color || undefined
    } else groups.set(key(tag), { tag, color: s.color || undefined, count: 1 })
  }
  const rank = (g: TagGroup) => {
    const i = TAG_PRESETS.findIndex((p) => p.tag === key(g.tag))
    return i < 0 ? TAG_PRESETS.length : i
  }
  return [...groups.values()].sort(
    (a, b) => rank(a) - rank(b) || a.tag.localeCompare(b.tag, undefined, { sensitivity: 'base' }),
  )
}

/**
 * The filter to apply: the chosen tag only while it is still in use, and only
 * once there is more than one tag to choose from. Otherwise no filtering, so a
 * tag that disappears never leaves an empty list behind.
 */
export function activeFilter(groups: readonly TagGroup[], chosen: string): string {
  if (groups.length < 2) return ''
  return groups.some((g) => key(g.tag) === key(chosen)) ? chosen : ''
}

/** Whether a server passes the tag filter; an empty filter keeps everything. */
export function matchesTag(s: Tagged, filter: string): boolean {
  return !filter || key(s.tag ?? '') === key(filter)
}
