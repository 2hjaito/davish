import { untrack } from 'svelte'

import { onSysmonTick, type SysmonTick, type Unsubscribe } from '../api/events'
import { monitorUnwatch, monitorWatch } from '../api/monitor'
import { hasRuntime } from '../api/runtime'
import { firstMockTick, nextMockTick } from './mockMetrics'

export type Metrics = Omit<SysmonTick, 'serverID'>

/** A tick and when it arrived (ms since epoch). */
export interface Sample extends Metrics {
  at: number
}

/** The longest range the Monitor charts show. */
export const HISTORY_MS = 15 * 60_000

export interface Series {
  latest: Sample
  /** Oldest first, spanning at most HISTORY_MS. */
  history: Sample[]
}

/**
 * Latest sysmon:tick and recent history per watched server. Only the menu
 * bar and Monitor windows watch, so the collector on the server runs only
 * while they need it. The menu bar watches the active server, so its
 * history is already there when a Monitor window opens.
 */
class MetricsStore {
  // Raw: a tick replaces the whole entry, and deep proxies over hundreds of
  // samples would cost more than they give.
  byServer = $state.raw<Record<string, Series>>({})

  #refs = new Map<string, number>()
  #off: Unsubscribe | undefined
  #mocks = new Map<string, ReturnType<typeof setInterval>>()

  latest(serverID: string): Sample | undefined {
    return this.byServer[serverID]?.latest
  }

  history(serverID: string): Sample[] {
    return this.byServer[serverID]?.history ?? []
  }

  /**
   * Starts collecting for serverID; call the returned function to stop.
   * Meant to be called from an $effect, so it reads no state the effect
   * would then depend on: starting fills byServer, which would re-run it.
   */
  watch(serverID: string): () => void {
    untrack(() => {
      const refs = this.#refs.get(serverID) ?? 0
      this.#refs.set(serverID, refs + 1)
      if (refs === 0) this.#start(serverID)
    })
    let done = false
    return () =>
      untrack(() => {
        if (done) return
        done = true
        const left = (this.#refs.get(serverID) ?? 1) - 1
        if (left > 0) {
          this.#refs.set(serverID, left)
          return
        }
        this.#refs.delete(serverID)
        this.#stop(serverID)
      })
  }

  /** Adds a tick for a watched server; exported for tests. */
  push(serverID: string, m: Metrics, at = Date.now()) {
    if (!this.#refs.has(serverID)) return // a tick can still be in flight after the last unwatch
    const sample = { ...m, at }
    const old = this.byServer[serverID]?.history ?? []
    const keepFrom = old.findIndex((s) => s.at > at - HISTORY_MS)
    const history = keepFrom < 0 ? [sample] : [...old.slice(keepFrom), sample]
    this.byServer = { ...this.byServer, [serverID]: { latest: sample, history } }
  }

  #start(serverID: string) {
    if (!hasRuntime) {
      this.#startMock(serverID)
      return
    }
    this.#off ??= onSysmonTick(({ serverID: id, ...m }) => this.push(id, m))
    void monitorWatch(serverID)
  }

  #stop(serverID: string) {
    // Stale numbers must not show up again when the server is watched later.
    const rest = { ...this.byServer }
    delete rest[serverID]
    this.byServer = rest
    if (!hasRuntime) {
      clearInterval(this.#mocks.get(serverID))
      this.#mocks.delete(serverID)
      return
    }
    void monitorUnwatch(serverID)
    if (this.#refs.size === 0) {
      this.#off?.()
      this.#off = undefined
    }
  }

  // A plain browser (vite dev, previews) has no Go side. Backfill a few
  // minutes so the charts have a shape straight away.
  #startMock(serverID: string) {
    let m = firstMockTick()
    const now = Date.now()
    for (let at = now - 5 * 60_000; at <= now; at += 2000) {
      this.push(serverID, m, at)
      m = nextMockTick(m)
    }
    const timer = setInterval(() => {
      const c = this.latest(serverID)
      if (c) this.push(serverID, nextMockTick(c))
    }, 2000)
    this.#mocks.set(serverID, timer)
  }
}

export const metrics = new MetricsStore()
