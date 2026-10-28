// Lays a commit list out as a graph: which lane each commit sits in and the
// lines joining it to its children above and parents below. Lanes never
// move sideways, so a branch keeps its column while it lasts.
import type { Commit } from '../../lib/api/git'

/** A line inside one row, from x1 at y1 to x2 at y2, in lane units. */
export interface Segment {
  from: number
  to: number
  /** 'top' joins the row's top edge to the node, 'bottom' the node to the bottom edge. */
  half: 'top' | 'bottom' | 'full'
  /** Lane whose color the line takes. */
  color: number
}

export interface GraphRow {
  lane: number
  segments: Segment[]
  /** Lanes in use in this row, for its width. */
  width: number
}

export function layout(commits: readonly Commit[]): GraphRow[] {
  // lanes[i] is the commit lane i waits for next; null when free.
  let lanes: (string | null)[] = []
  const rows: GraphRow[] = []
  for (const c of commits) {
    const before = lanes
    let lane = before.indexOf(c.hash)
    if (lane < 0) {
      // A branch tip: nothing above waits for it.
      lane = before.indexOf(null)
      if (lane < 0) lane = before.length
    }
    const after = before.slice()
    const segments: Segment[] = []
    before.forEach((h, j) => {
      if (h === null) return
      if (h === c.hash) {
        // Children reach the node; every lane waiting for it ends here.
        segments.push({ from: j, to: lane, half: 'top', color: j })
        after[j] = null
      } else {
        segments.push({ from: j, to: j, half: 'full', color: j })
      }
    })
    after[lane] = null
    c.parents.forEach((p, n) => {
      // The first parent always continues the node's lane, even when another
      // lane already waits for it: both lanes join at the parent. A merged
      // parent joins the lane waiting for it, or takes a free one.
      let k = n === 0 ? lane : after.indexOf(p)
      if (k < 0) {
        k = after.indexOf(null)
        if (k < 0) k = after.length
      }
      after[k] = p
      segments.push({ from: lane, to: k, half: 'bottom', color: n === 0 ? lane : k })
    })
    while (after.length > 0 && after[after.length - 1] === null) after.pop()
    rows.push({ lane, segments, width: Math.max(before.length, after.length, lane + 1) })
    lanes = after
  }
  return rows
}
