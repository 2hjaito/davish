// The folder tree of a mount, from the overview's folders (three levels
// deep) and the folders measured since, one level at a time.
import type { DiskEntry, Overview, Usage } from '../../lib/api/disk'

/** How deep below the mount the overview lists folders (internal/disk). */
export const OVERVIEW_DEPTH = 3

export interface Row {
  path: string
  name: string
  size: number
  depth: number
  /** Size of the parent, for the share bar. */
  parentSize: number
  /** Its children are known: they show when expanded. */
  known: boolean
  hasChildren: boolean
}

export const parentOf = (p: string) => p.slice(0, p.lastIndexOf('/')) || '/'
export const nameOf = (p: string) => p.slice(p.lastIndexOf('/') + 1) || p

const depthOf = (root: string, p: string) =>
  p === root ? 0 : p.slice(root === '/' ? 0 : root.length).split('/').length - 1

/** Children by parent, largest first. */
export function buildTree(
  dirs: DiskEntry[],
  measured: Record<string, Usage>,
): Map<string, DiskEntry[]> {
  const byPath = new Map<string, DiskEntry>()
  for (const d of dirs) byPath.set(d.path, d)
  // A later measurement is fresher than the overview.
  for (const u of Object.values(measured)) for (const e of u.entries) byPath.set(e.path, e)
  const tree = new Map<string, DiskEntry[]>()
  for (const e of byPath.values()) {
    const p = parentOf(e.path)
    const kids = tree.get(p)
    if (kids) kids.push(e)
    else tree.set(p, [e])
  }
  for (const kids of tree.values())
    kids.sort((a, b) => b.size - a.size || a.path.localeCompare(b.path))
  return tree
}

/** Whether the children of p are in the tree: within the overview's depth, or measured. */
export function knownChildren(root: string, p: string, measured: Record<string, Usage>): boolean {
  return depthOf(root, p) < OVERVIEW_DEPTH || p in measured
}

/** The rows to show: root's children, and the children of expanded folders under them. */
export function visibleRows(
  root: string,
  rootSize: number,
  tree: Map<string, DiskEntry[]>,
  measured: Record<string, Usage>,
  expanded: ReadonlySet<string>,
): Row[] {
  const rows: Row[] = []
  const walk = (parent: string, parentSize: number, depth: number) => {
    for (const e of tree.get(parent) ?? []) {
      const known = knownChildren(root, e.path, measured)
      const hasChildren = (tree.get(e.path)?.length ?? 0) > 0
      rows.push({
        path: e.path,
        name: nameOf(e.path),
        size: e.size,
        depth,
        parentSize,
        known,
        hasChildren,
      })
      if (expanded.has(e.path)) walk(e.path, e.size, depth + 1)
    }
  }
  walk(root, rootSize, 0)
  return rows
}

/** Every known folder whose path contains q, largest first, for the search box. */
export function matchingRows(root: string, tree: Map<string, DiskEntry[]>, q: string): Row[] {
  const needle = q.trim().toLowerCase()
  const rows: Row[] = []
  for (const [parent, kids] of tree) {
    const parentSize = tree.get(parentOf(parent))?.find((e) => e.path === parent)?.size ?? 0
    for (const e of kids) {
      if (!e.path.startsWith(root === '/' ? '/' : root + '/')) continue
      if (!nameOf(e.path).toLowerCase().includes(needle)) continue
      rows.push({
        path: e.path,
        name: e.path,
        size: e.size,
        depth: 0,
        parentSize: parentSize || e.size,
        known: true,
        hasChildren: false,
      })
    }
  }
  return rows.sort((a, b) => b.size - a.size)
}

/**
 * The overview after path was deleted or emptied: it leaves the lists and
 * its size leaves every folder above it and the disk's use, so the app
 * need not scan the whole disk again.
 */
export function withoutPath(o: Overview, path: string, size: number): Overview {
  const gone = (p: string) => p === path || p.startsWith(path + '/')
  const above = (p: string) => path.startsWith(p === '/' ? '/' : p + '/')
  const shrink = (e: DiskEntry) => (above(e.path) ? { ...e, size: Math.max(0, e.size - size) } : e)
  return {
    ...o,
    mount: { ...o.mount, used: Math.max(0, o.mount.used - size) },
    measured: Math.max(0, o.measured - size),
    dirs: o.dirs.filter((e) => !gone(e.path)).map(shrink),
    files: o.files.filter((e) => !gone(e.path)),
  }
}
