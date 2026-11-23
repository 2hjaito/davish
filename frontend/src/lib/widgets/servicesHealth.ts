// What the Services widget makes of `systemctl list-units` (ADR 0026), apart
// from reading it. Every unit falls in one group, so the parts of the bar add
// up to the total.
import type { Unit } from '../api/services'

export type Group = 'running' | 'exited' | 'starting' | 'bad' | 'inactive'

/**
 * The group of a unit: failed or retrying after a crash (auto-restart) is
 * bad; still activating is starting; active and running is running; active
 * but not running (a oneshot that is done) is exited; the rest is inactive.
 */
export function groupOf(u: Pick<Unit, 'active' | 'sub'>): Group {
  if (u.active === 'failed' || u.sub === 'failed' || u.sub === 'auto-restart') return 'bad'
  if (u.active === 'activating' || u.active === 'reloading') return 'starting'
  if (u.active === 'active') return u.sub === 'running' ? 'running' : 'exited'
  return 'inactive'
}

export interface Problem {
  /** The unit without its ".service". */
  name: string
  why: 'failed' | 'restarting'
}

export interface ServicesSummary {
  total: number
  counts: Record<Group, number>
  /** The units that are bad, up to LIST_MAX, in systemctl's order. */
  problems: Problem[]
  /** How many more are bad than are named. */
  more: number
}

export const LIST_MAX = 20

const short = (name: string) => name.replace(/\.service$/, '')

export function summarize(list: readonly Unit[]): ServicesSummary {
  const counts: Record<Group, number> = { running: 0, exited: 0, starting: 0, bad: 0, inactive: 0 }
  const problems: Problem[] = []
  let more = 0
  for (const u of list) {
    const g = groupOf(u)
    counts[g]++
    if (g !== 'bad') continue
    if (problems.length < LIST_MAX)
      problems.push({
        name: short(u.name),
        why: u.sub === 'auto-restart' ? 'restarting' : 'failed',
      })
    else more++
  }
  return { total: list.length, counts, problems, more }
}

/** How many units are bad, those past the list included. */
export const problemCount = (s: ServicesSummary) => s.counts.bad

export const hasProblem = (s: ServicesSummary) => s.counts.bad > 0
