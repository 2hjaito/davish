// Pure helpers behind the Files app: ordering, filtering, paths and the
// virtual list window.
import type { Entry } from '../../lib/api/files'

export type SortKey = 'name' | 'size' | 'modified'

export interface SortOrder {
  key: SortKey
  desc: boolean
}

export const SORT_KEYS: readonly SortKey[] = ['name', 'size', 'modified']

// "file2" before "file10", and case does not split the list in two.
const collator = new Intl.Collator('vi', { numeric: true, sensitivity: 'base' })

const byName = (a: Entry, b: Entry) =>
  collator.compare(a.name, b.name) || (a.name < b.name ? -1 : a.name > b.name ? 1 : 0)

/**
 * Folders first, like most server file managers, then by key. Ties fall back
 * to the name, A to Z, so the order is stable. Folders have no size, so a
 * size order keeps them A to Z too.
 */
export function sortEntries(entries: readonly Entry[], order: SortOrder): Entry[] {
  const dir = order.desc ? -1 : 1
  return [...entries].sort((a, b) => {
    if (a.dir !== b.dir) return a.dir ? -1 : 1
    let c = 0
    if (order.key === 'name') c = byName(a, b) * dir
    else if (order.key === 'size' && !a.dir) c = (a.size - b.size) * dir
    else if (order.key === 'modified') c = (a.modTime - b.modTime) * dir
    return c || byName(a, b)
  })
}

/** Clicking the current column flips it; a new column starts the way people expect. */
export function nextOrder(current: SortOrder, key: SortKey): SortOrder {
  if (current.key === key) return { key, desc: !current.desc }
  // Biggest and newest are what one looks for first.
  return { key, desc: key !== 'name' }
}

export function filterEntries(entries: readonly Entry[], query: string): readonly Entry[] {
  const q = query.trim().toLocaleLowerCase('vi')
  if (!q) return entries
  return entries.filter((e) => e.name.toLocaleLowerCase('vi').includes(q))
}

export function joinPath(dir: string, name: string): string {
  return dir === '/' ? `/${name}` : `${dir}/${name}`
}

export function parentPath(path: string): string {
  const i = path.lastIndexOf('/')
  return i <= 0 ? '/' : path.slice(0, i)
}

export function baseName(path: string): string {
  return path === '/' ? '/' : path.slice(path.lastIndexOf('/') + 1)
}

export interface Crumb {
  name: string
  path: string
}

/** The folders from the root down to path, root excluded. */
export function crumbs(path: string): Crumb[] {
  return path
    .split('/')
    .filter(Boolean)
    .map((name, i, all) => ({ name, path: '/' + all.slice(0, i + 1).join('/') }))
}

/** Whether path is place itself or somewhere below it. */
export function within(path: string, place: string): boolean {
  return path === place || path.startsWith(place === '/' ? '/' : place + '/')
}

export interface DateWords {
  today: string
  yesterday: string
}

/** Like Finder: "Hôm nay, 14:03", "Hôm qua, 09:12", otherwise "12/10/2026". */
export function formatModTime(unix: number, now: Date, words: DateWords): string {
  const d = new Date(unix * 1000)
  const pad = (n: number) => String(n).padStart(2, '0')
  const time = `${pad(d.getHours())}:${pad(d.getMinutes())}`
  const day = (x: Date) => new Date(x.getFullYear(), x.getMonth(), x.getDate()).getTime()
  const days = Math.round((day(now) - day(d)) / 86_400_000)
  if (days === 0) return `${words.today}, ${time}`
  if (days === 1) return `${words.yesterday}, ${time}`
  return `${pad(d.getDate())}/${pad(d.getMonth() + 1)}/${d.getFullYear()}`
}

export interface RowWindow {
  start: number
  end: number
}

/** The rows to render for a scroll position, with overscan on both sides. */
export function visibleRows(
  scrollTop: number,
  viewport: number,
  rowHeight: number,
  count: number,
  overscan = 8,
): RowWindow {
  const first = Math.floor(Math.max(0, scrollTop) / rowHeight)
  const last = Math.ceil((Math.max(0, scrollTop) + viewport) / rowHeight)
  return { start: Math.max(0, first - overscan), end: Math.min(count, last + overscan) }
}

/** The scrollTop that brings row index into view, or null when it already is. */
export function scrollToRow(
  index: number,
  scrollTop: number,
  viewport: number,
  rowHeight: number,
): number | null {
  const top = index * rowHeight
  if (top < scrollTop) return top
  if (top + rowHeight > scrollTop + viewport) return top + rowHeight - viewport
  return null
}
