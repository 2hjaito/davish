import { dockerList } from '../api/docker'
import type { Container } from '../api/docker'
import { toUserError } from '../api/errors'
import { hasRuntime } from '../api/runtime'
import { type DockerSummary, summarize } from '../widgets/dockerHealth'
import { pageVisible } from '../wm/pageVisible.svelte'
import { servers } from './servers.svelte'

/** Why a server has no reading: docker is missing, needs root, or could not be read. */
export type Problem = 'missing' | 'needsRoot' | 'failed'

export interface Reading {
  summary: DockerSummary | null
  /** When the summary was read, ms since epoch; 0 before the first. */
  at: number
  problem: Problem | null
  loading: boolean
}

const EMPTY: Reading = { summary: null, at: 0, problem: null, loading: false }

/** How often a watched server is read while Davish is on screen and in front. */
export const REFRESH_MS = 60_000
/** Two reads of one server are never closer than this, whatever asks. */
export const MIN_GAP_MS = 10_000

export interface Deps {
  list: (serverID: string) => Promise<Container[]>
  connected: (serverID: string) => boolean
  /** Davish is visible and has the focus: the only time a timer reads anything. */
  active: () => boolean
  now: () => number
}

const NEEDS_ROOT = new Set(['SUDO_PASSWORD_REQUIRED', 'SUDO_PASSWORD_WRONG', 'CMD_NEEDS_ROOT'])

/**
 * The reading the Docker health widgets share (ADR 0026): one `docker ps -a`
 * per watched server, however many widgets watch it. Docker and Services do
 * not poll, so that their every list is not a line in the command log; this
 * is the one exception, and it is slow, only while someone looks, and a
 * server that says docker is missing or needs root is not asked again until
 * Davish comes back to the front. A list is a read, so the command log hides
 * it behind "Changes only". It never asks for a sudo password.
 */
export class DockerHealth {
  /** Raw: a read replaces the whole entry. */
  byServer = $state.raw<Record<string, Reading>>({})

  // Bookkeeping that no template or effect reads: the reading is byServer.
  #deps: Deps
  // eslint-disable-next-line svelte/prefer-svelte-reactivity -- bookkeeping, never read by a template
  #refs = new Map<string, number>()
  // eslint-disable-next-line svelte/prefer-svelte-reactivity -- bookkeeping, never read by a template
  #last = new Map<string, number>()
  // eslint-disable-next-line svelte/prefer-svelte-reactivity -- bookkeeping, never read by a template
  #inflight = new Set<string>()
  /** Servers whose last problem is not worth another try on the timer. */
  // eslint-disable-next-line svelte/prefer-svelte-reactivity -- bookkeeping, never read by a template
  #blocked = new Set<string>()
  #timer: ReturnType<typeof setInterval> | undefined
  #onFocus = () => this.onFocus()

  constructor(deps: Deps) {
    this.#deps = deps
  }

  of(serverID: string): Reading {
    return this.byServer[serverID] ?? EMPTY
  }

  /**
   * Starts reading serverID on a timer; call the result to stop. Meant for an
   * $effect: it reads no state of its own. Reading the first time is up to
   * the caller, with refresh.
   */
  watch(serverID: string): () => void {
    this.#refs.set(serverID, (this.#refs.get(serverID) ?? 0) + 1)
    if (this.#refs.size === 1 && (this.#refs.get(serverID) ?? 0) === 1) this.#start()
    let done = false
    return () => {
      if (done) return
      done = true
      const left = (this.#refs.get(serverID) ?? 1) - 1
      if (left > 0) this.#refs.set(serverID, left)
      else this.#refs.delete(serverID)
      if (this.#refs.size === 0) this.#stop()
    }
  }

  /** Reads serverID now, unless it is not connected, a read is under way or one just ran. */
  async refresh(serverID: string): Promise<void> {
    if (!this.#deps.connected(serverID) || this.#inflight.has(serverID)) return
    const now = this.#deps.now()
    if (now - (this.#last.get(serverID) ?? -Infinity) < MIN_GAP_MS) return
    this.#last.set(serverID, now)
    this.#inflight.add(serverID)
    const before = this.of(serverID)
    this.#set(serverID, { ...before, loading: true })
    try {
      const list = await this.#deps.list(serverID)
      this.#blocked.delete(serverID)
      this.#set(serverID, {
        summary: summarize(list),
        at: this.#deps.now(),
        problem: null,
        loading: false,
      })
    } catch (e) {
      const code = toUserError(e).code
      const problem: Problem =
        code === 'CMD_MISSING' ? 'missing' : NEEDS_ROOT.has(code) ? 'needsRoot' : 'failed'
      if (problem !== 'failed') this.#blocked.add(serverID)
      // What was read before stays: a failed read does not blank the widget.
      this.#set(serverID, { ...before, problem, loading: false })
    } finally {
      this.#inflight.delete(serverID)
    }
  }

  /** Davish came to the front: every watched server may be read again, blocked or not. */
  onFocus() {
    this.#blocked.clear()
    for (const id of this.#refs.keys()) void this.refresh(id)
  }

  #tick() {
    if (!this.#deps.active()) return
    for (const id of this.#refs.keys()) if (!this.#blocked.has(id)) void this.refresh(id)
  }

  #start() {
    this.#timer = setInterval(() => this.#tick(), REFRESH_MS)
    if (typeof window !== 'undefined') window.addEventListener('focus', this.#onFocus)
  }

  #stop() {
    clearInterval(this.#timer)
    this.#timer = undefined
    if (typeof window !== 'undefined') window.removeEventListener('focus', this.#onFocus)
  }

  #set(serverID: string, r: Reading) {
    this.byServer = { ...this.byServer, [serverID]: r }
  }
}

/** Sample containers for the browser preview, which has no server. */
async function demoList(): Promise<Container[]> {
  const c = (name: string, state: string, status: string): Container => ({
    id: name,
    name,
    image: 'demo',
    state,
    status,
    ports: '',
    command: '',
    created: '',
    runningFor: '',
  })
  return [
    c('web', 'running', 'Up 3 days (healthy)'),
    c('api', 'running', 'Up 3 days (healthy)'),
    c('db', 'running', 'Up 3 days (healthy)'),
    c('cache', 'running', 'Up 2 hours (unhealthy)'),
    c('worker', 'restarting', 'Restarting (1) 12 seconds ago'),
    c('backup', 'exited', 'Exited (0) 6 hours ago'),
  ]
}

export const dockerHealth = new DockerHealth({
  list: hasRuntime ? dockerList : demoList,
  connected: (id) => !hasRuntime || servers.list.find((s) => s.id === id)?.state === 'connected',
  active: () => pageVisible.visible && typeof document !== 'undefined' && document.hasFocus(),
  now: Date.now,
})
