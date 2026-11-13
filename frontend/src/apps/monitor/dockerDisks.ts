// Which mount point holds Docker's data and each container's volumes and
// bind mounts, for the Disks card on Monitor's Info page.
import type { DiskUse } from '../../lib/api/docker'

/** The longest mount point that contains path; "/" when nothing else does. */
export function mountOf(path: string, mounts: readonly string[]): string | undefined {
  let best: string | undefined
  for (const m of mounts) {
    const inside = m === '/' || path === m || path.startsWith(m.endsWith('/') ? m : `${m}/`)
    if (inside && (!best || m.length > best.length)) best = m
  }
  return best
}

export interface MountUse {
  /** Docker's own data folder lives here. */
  root: boolean
  /** Compose projects, or lone containers, with data here, once each. */
  owners: string[]
}

export function useByMount(use: DiskUse, mounts: readonly string[]): Map<string, MountUse> {
  const out = new Map<string, MountUse>()
  const at = (m: string) => {
    let u = out.get(m)
    if (!u) out.set(m, (u = { root: false, owners: [] }))
    return u
  }
  const root = use.rootDir ? mountOf(use.rootDir, mounts) : undefined
  if (root) at(root).root = true
  for (const x of use.mounts) {
    const m = mountOf(x.source, mounts)
    if (!m) continue
    const owner = x.project || x.container
    const u = at(m)
    if (!u.owners.includes(owner)) u.owners.push(owner)
  }
  for (const u of out.values()) u.owners.sort()
  return out
}
