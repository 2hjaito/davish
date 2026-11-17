// Running one command on several servers (docs/adr/0021): what each tab
// shows, and when a tab waits for the user to type.
import type { FleetJob, JobState } from '../../lib/api/fleet'

/** How long a run stays quiet, mid-line, before its tab counts as waiting. */
export const IDLE_MS = 1500

/** How much of the end of the output is kept to spot a prompt. */
const TAIL = 400

// Escape sequences: CSI, OSC and two-byte ones; none ends a prompt.
// eslint-disable-next-line no-control-regex -- matching terminal escapes is the point
const ESCAPES = /\x1b\[[0-9;?]*[ -/]*[@-~]|\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)|\x1b[@-_]/g

/** Prompts that wait for an answer, matched at the end of the last line. */
const PROMPTS = [
  /\[sudo\] password for [^:]*:\s*$/i,
  /password[^:\n]{0,40}:\s*$/i,
  /passphrase[^:\n]{0,60}:\s*$/i,
  /\[y\/n\]\s*\??\s*$/i,
  /\(y\/n\)\s*\??\s*$/i,
  /\(yes\/no(?:\/\[fingerprint\])?\)\s*\??\s*$/i,
  /do you want to continue\??\s*\[?[yn/]*\]?\s*$/i,
  /is this ok \[y\/n\]:\s*$/i,
]

/** What sudo -n prints when it would have to ask for a password. */
const SUDO_REFUSED = /sudo: (?:a password is required|a terminal is required)/

/** The last line of output as it reads on screen, escapes removed. */
export function lastLine(tail: string): string {
  const text = tail.replace(ESCAPES, '')
  const line = text.slice(text.lastIndexOf('\n') + 1)
  // A carriage return rewrites the line from its start.
  const parts = line.split('\r').filter((p) => p !== '')
  return parts.at(-1) ?? ''
}

/** Keeps the end of the output, enough to read the last line. */
export const keepTail = (tail: string, more: string): string => (tail + more).slice(-TAIL)

/** The output ends in a known prompt. */
export const promptShown = (tail: string): boolean => {
  const line = lastLine(tail)
  return line.trim() !== '' && PROMPTS.some((p) => p.test(line))
}

/**
 * Whether a running job waits for input: a known prompt at once, or a
 * line left unfinished and quiet for IDLE_MS.
 */
export function waiting(state: JobState, tail: string, quietMs: number): boolean {
  if (state !== 'running') return false
  if (promptShown(tail)) return true
  const text = tail.replace(ESCAPES, '')
  return quietMs >= IDLE_MS && !text.endsWith('\n') && lastLine(tail).trim() !== ''
}

/** sudo -n gave up: the server wants a password Davish never types for a run. */
export const sudoRefused = (text: string): boolean => SUDO_REFUSED.test(text)

/** Counts of the jobs by state, for the summary line. */
export function tally(jobs: readonly Pick<FleetJob, 'state'>[]): Record<JobState, number> {
  const t: Record<JobState, number> = { queued: 0, running: 0, done: 0, failed: 0, stopped: 0 }
  for (const j of jobs) t[j.state]++
  return t
}

/** The run is over: nothing queued or running. */
export const finished = (jobs: readonly Pick<FleetJob, 'state'>[]): boolean =>
  jobs.every((j) => j.state !== 'queued' && j.state !== 'running')

/** The tab to bring forward: the first waiting one, unless the shown one waits too. */
export function nextActive(
  order: readonly string[],
  isWaiting: (serverID: string) => boolean,
  active: string,
): string {
  if (isWaiting(active)) return active
  return order.find(isWaiting) ?? active
}

/** A scripted run for the browser preview, which has no servers to reach. */
export interface MockStep {
  serverID: string
  at: number
  out?: string
  state?: JobState
  exit?: number
}

export function mockSteps(ids: readonly string[], command: string): MockStep[] {
  const steps: MockStep[] = []
  ids.forEach((id, i) => {
    const t0 = 150 + i * 120
    steps.push({ serverID: id, at: t0, state: 'running' })
    if (i === 1 && command.includes('sudo')) {
      steps.push({ serverID: id, at: t0 + 300, out: `[sudo] password for deploy: ` })
      return
    }
    steps.push({
      serverID: id,
      at: t0 + 250,
      out: ` ${String(10 + i).padStart(2, '0')}:42:07 up ${12 + i * 7} days,  2 users,  load average: 0.${i + 1}2, 0.18, 0.11\r\n`,
    })
    steps.push({
      serverID: id,
      at: t0 + 400,
      state: i === 2 ? 'failed' : 'done',
      exit: i === 2 ? 1 : 0,
    })
  })
  return steps
}
