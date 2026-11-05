// Deploy runs a terminal is showing, read from the progress marks the
// deploy line prints (internal/git/deploy.go, docs/adr/0012). Marks come
// from server output, which anyone on the server can write: they only
// draw progress, are taken only with the nonce of a run being watched,
// and never decide what runs.
import { t } from '../i18n'
import { focusTerminal } from '../wm/intents.svelte'
import { notifications } from './notifications.svelte'
import { load, save } from './persist'
import { servers } from './servers.svelte'

/** The OSC number the marks use; the terminal swallows these. */
export const DEPLOY_OSC = 7761

export type Mark =
  | { nonce: string; kind: 'begin'; steps: number }
  | { nonce: string; kind: 'step'; step: number }
  | { nonce: string; kind: 'end'; exit: number; commit?: string }

const int = (s: string | undefined) => (s !== undefined && /^\d{1,5}$/.test(s) ? Number(s) : NaN)

/** Reads the data of an OSC 7761 sequence; null for anything else. */
export function parseMark(data: string): Mark | null {
  const [tag, nonce, kind, a, b] = data.split(';')
  if (tag !== 'davish-deploy' || !nonce || !/^[0-9a-f]{1,64}$/.test(nonce)) return null
  const n = int(a)
  if (Number.isNaN(n)) return null
  switch (kind) {
    case 'begin':
      return { nonce, kind, steps: n }
    case 'step':
      return n >= 1 ? { nonce, kind, step: n } : null
    case 'end':
      return { nonce, kind, exit: n, commit: b && /^[0-9a-f]{4,40}$/.test(b) ? b : undefined }
  }
  return null
}

export type RunState = 'waiting' | 'running' | 'ok' | 'failed' | 'stopped'

export interface DeployRun {
  nonce: string
  /** A deploy has steps; a script from the desktop is one long step. */
  kind: 'deploy' | 'script'
  serverID: string
  dir: string
  steps: string[]
  /** waiting: typed but not yet run; stopped: Ctrl-C or the tab closed. */
  state: RunState
  /** Davish presses Enter itself; otherwise the user does (sudo). */
  auto: boolean
  /** The 1-based step running, or the one that failed or was stopped; 0 before the first. */
  current: number
  /** When each step started, ms since epoch. */
  stepAt: number[]
  startedAt?: number
  endedAt?: number
  exit?: number
  commit?: string
}

/** A finished run, kept on this machine without its output. */
export interface DeployRecord {
  at: number
  state: 'ok' | 'failed' | 'stopped'
  /** The step that failed or was stopped. */
  step?: number
  exit?: number
  commit?: string
  /** How long each step that ran took, ms. */
  durations: number[]
}

const HISTORY = 'deploy-history'
export const HISTORY_MAX = 20
const historyKey = (serverID: string, dir: string) => `${serverID}:${dir}`

/** Step durations of a run, the last one ending at end. */
export function durations(run: DeployRun, end: number): number[] {
  return run.stepAt.map((at, i) => (run.stepAt[i + 1] ?? end) - at)
}

export function record(run: DeployRun): DeployRecord {
  const state = run.state === 'ok' || run.state === 'failed' ? run.state : 'stopped'
  return {
    at: run.startedAt ?? run.endedAt ?? 0,
    state,
    step: state === 'ok' ? undefined : run.current,
    exit: run.exit,
    commit: run.commit,
    durations: durations(run, run.endedAt ?? 0),
  }
}

const repoName = (dir: string) => dir.split('/').filter(Boolean).pop() ?? dir

class Deploys {
  /** Runs watched in open terminals, by nonce. */
  runs = $state<Record<string, DeployRun>>({})
  /** Finished runs per server and folder, newest first. */
  history = $state<Record<string, DeployRecord[]>>(load(HISTORY, {}))
  /** Press Ctrl-C in the session of a run, by nonce; set by its terminal tab. */
  #interrupts = new Map<string, () => void>()

  /** Starts following a deploy line about to be typed into a terminal. */
  watch(
    serverID: string,
    dir: string,
    nonce: string,
    steps: string[],
    auto = false,
    kind: DeployRun['kind'] = 'deploy',
  ) {
    this.runs[nonce] = {
      nonce,
      kind,
      serverID,
      dir,
      steps,
      state: 'waiting',
      auto,
      current: 0,
      stepAt: [],
    }
  }

  historyOf(serverID: string, dir: string): DeployRecord[] {
    return this.history[historyKey(serverID, dir)] ?? []
  }

  /** The run going for dir on serverID, if any. */
  active(serverID: string, dir: string): DeployRun | undefined {
    return Object.values(this.runs).find(
      (r) =>
        r.serverID === serverID &&
        r.dir === dir &&
        (r.state === 'waiting' || r.state === 'running'),
    )
  }

  /** The terminal tab showing a run says how to interrupt it. */
  onInterrupt(nonce: string, interrupt: () => void) {
    this.#interrupts.set(nonce, interrupt)
  }

  /** Stops a run from outside its terminal: the Stop button, the desktop icon. */
  interrupt(nonce: string) {
    this.#interrupts.get(nonce)?.()
  }

  /** Applies a mark, if it belongs to a run still going. */
  apply(m: Mark, now = Date.now()) {
    const run = this.runs[m.nonce]
    if (!run || (run.state !== 'waiting' && run.state !== 'running')) return
    if (run.state === 'waiting') {
      run.state = 'running'
      run.startedAt = now
    }
    if (m.kind === 'step') {
      // A step only moves forward; a repeat or a skip back is not ours.
      if (m.step !== run.current + 1 || m.step > run.steps.length) return
      run.current = m.step
      run.stepAt.push(now)
    } else if (m.kind === 'end') {
      run.exit = m.exit
      run.commit = m.commit
      this.#finish(run, m.exit === 0 && run.current === run.steps.length ? 'ok' : 'failed', now)
    }
  }

  /** Ctrl-C or a closed tab. A run never started is just forgotten. */
  stop(nonce: string, now = Date.now()) {
    const run = this.runs[nonce]
    if (!run) return
    if (run.state === 'waiting') delete this.runs[nonce]
    else if (run.state === 'running') this.#finish(run, 'stopped', now)
  }

  /** Drops a finished run from view, when its tab closes. */
  forget(nonce: string) {
    const run = this.runs[nonce]
    if (run && run.state !== 'running') delete this.runs[nonce]
    this.#interrupts.delete(nonce)
  }

  #finish(run: DeployRun, state: 'ok' | 'failed' | 'stopped', now: number) {
    run.state = state
    run.endedAt = now
    const key = historyKey(run.serverID, run.dir)
    this.history[key] = [record(run), ...(this.history[key] ?? [])].slice(0, HISTORY_MAX)
    save(HISTORY, this.history)
    this.#notify(run)
  }

  #notify(run: DeployRun) {
    const server = servers.list.find((s) => s.id === run.serverID)?.name ?? run.serverID
    const d = t.deploys
    if (run.kind === 'script') {
      this.#notifyScript(run, server)
      return
    }
    const name = repoName(run.dir)
    const step = run.steps[run.current - 1] ?? ''
    notifications.add({
      id: `deploy:${run.nonce}`,
      level: run.state === 'ok' ? 'info' : run.state === 'failed' ? 'error' : 'warn',
      title:
        run.state === 'ok'
          ? d.doneTitle(name, server)
          : run.state === 'failed'
            ? d.failedTitle(name, server)
            : d.stoppedTitle(name, server),
      body:
        run.state === 'ok'
          ? d.doneBody(run.steps.length, run.commit)
          : d.failedBody(run.current, step, run.exit),
      serverID: run.serverID,
      at: run.endedAt ?? Date.now(),
      action: { label: d.showTerminal, run: () => focusTerminal(run.serverID) },
    })
  }

  #notifyScript(run: DeployRun, server: string) {
    const s = t.deploys.script
    const name = run.steps[0] ?? repoName(run.dir)
    const texts = {
      ok: [s.doneTitle(name, server), s.doneBody],
      failed: [s.failedTitle(name, server), run.current === 0 ? s.noDir : s.exitBody(run.exit)],
      stopped: [s.stoppedTitle(name, server), s.stoppedBody],
    } as const
    const [title, body] =
      texts[run.state === 'ok' || run.state === 'failed' ? run.state : 'stopped']
    notifications.add({
      id: `deploy:${run.nonce}`,
      level: run.state === 'ok' ? 'info' : run.state === 'failed' ? 'error' : 'warn',
      title,
      body,
      serverID: run.serverID,
      at: run.endedAt ?? Date.now(),
      action: { label: t.deploys.showTerminal, run: () => focusTerminal(run.serverID) },
    })
  }
}

export const deploys = new Deploys()
