// Matching and ranking for Spotlight (⌘K): what the user typed against
// apps, services, containers and folders, plus paths and commands.

/**
 * How well query matches text: 0 when it does not. Every query character
 * must appear in order; a prefix beats a word start, which beats letters
 * spread out, and shorter texts win ties.
 */
export function score(query: string, text: string): number {
  const q = query.trim().toLowerCase()
  const s = text.toLowerCase()
  if (!q) return 1
  if (s.startsWith(q)) return 1000 - s.length
  const at = s.indexOf(q)
  if (at >= 0) return (/[\s/._@-]/.test(s[at - 1] ?? ' ') ? 800 : 600) - at - s.length / 10
  // Letters in order, each gap costing a little.
  let pos = -1
  let gaps = 0
  for (const c of q) {
    const next = s.indexOf(c, pos + 1)
    if (next < 0) return 0
    gaps += next - pos - 1
    pos = next
  }
  return Math.max(1, 300 - gaps * 5 - s.length / 10)
}

export type Kind =
  'app' | 'path' | 'command' | 'service' | 'container' | 'program' | 'folder' | 'help'

export interface Candidate {
  kind: Kind
  /** Unique within its kind. */
  id: string
  label: string
  /** Matched too, with less weight: a description, an image. */
  detail?: string
}

/** The order groups appear in, and how many each shows. */
export const GROUPS: { kind: Kind; max: number }[] = [
  { kind: 'path', max: 1 },
  { kind: 'command', max: 1 },
  { kind: 'app', max: 4 },
  { kind: 'service', max: 5 },
  { kind: 'container', max: 5 },
  { kind: 'program', max: 4 },
  { kind: 'folder', max: 3 },
  { kind: 'help', max: 2 },
]

/** The candidates matching query, grouped in GROUPS order, best first within a group. */
export function rank<T extends Candidate>(candidates: readonly T[], query: string): T[] {
  const scored = candidates
    .map((c) => ({
      c,
      // Details match only as a whole word or phrase: letters scattered
      // over a long description would match almost anything.
      s: Math.max(
        score(query, c.label),
        c.detail && c.detail.toLowerCase().includes(query.trim().toLowerCase())
          ? score(query, c.detail) * 0.6
          : 0,
      ),
    }))
    .filter((x) => x.s > 0)
  const out: T[] = []
  for (const g of GROUPS) {
    out.push(
      ...scored
        .filter((x) => x.c.kind === g.kind)
        .sort((a, b) => b.s - a.s)
        .slice(0, g.max)
        .map((x) => x.c),
    )
  }
  return out
}

/** What the query asks for beyond a search: a path to open or a command to type. */
export function intent(
  query: string,
): { kind: 'path'; path: string } | { kind: 'command'; command: string } | null {
  const q = query.trim()
  if (q.startsWith('/')) {
    const path = q.replace(/\/{2,}/g, '/').replace(/(.)\/$/, '$1')
    return path.split('/').includes('..') ? null : { kind: 'path', path }
  }
  if (q.startsWith('>')) {
    const command = q.slice(1).trim()
    return command ? { kind: 'command', command } : null
  }
  return null
}
