import { onSysmonTick, type SysmonTick, type Unsubscribe } from '../api/events'
import { monitorUnwatch, monitorWatch } from '../api/monitor'
import { hasRuntime } from '../api/runtime'
import { firstMockTick, nextMockTick } from './mockMetrics'

export type Metrics = Omit<SysmonTick, 'serverID'>

/**
 * Latest sysmon:tick per watched server. Only the menu bar and Monitor
 * windows watch, so the collector on the server runs only while they need it.
 */
class MetricsStore {
  byServer = $state<Record<string, Metrics>>({})

  #refs = new Map<string, number>()
  #off: Unsubscribe | undefined
  #mocks = new Map<string, ReturnType<typeof setInterval>>()

  /** Starts collecting for serverID; call the returned function to stop. */
  watch(serverID: string): () => void {
    const refs = this.#refs.get(serverID) ?? 0
    this.#refs.set(serverID, refs + 1)
    if (refs === 0) this.#start(serverID)
    let done = false
    return () => {
      if (done) return
      done = true
      const left = (this.#refs.get(serverID) ?? 1) - 1
      if (left > 0) {
        this.#refs.set(serverID, left)
        return
      }
      this.#refs.delete(serverID)
      this.#stop(serverID)
    }
  }

  #start(serverID: string) {
    if (!hasRuntime) {
      this.#startMock(serverID)
      return
    }
    this.#off ??= onSysmonTick((t) => this.#apply(t))
    void monitorWatch(serverID)
  }

  #stop(serverID: string) {
    // Stale numbers must not show up again when the server is watched later.
    delete this.byServer[serverID]
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

  #apply({ serverID, ...m }: SysmonTick) {
    // A tick can still be in flight after the last unwatch.
    if (this.#refs.has(serverID)) this.byServer[serverID] = m
  }

  // A plain browser (vite dev, previews) has no Go side.
  #startMock(serverID: string) {
    this.byServer[serverID] = firstMockTick()
    const timer = setInterval(() => {
      const c = this.byServer[serverID]
      if (c) this.byServer[serverID] = nextMockTick(c)
    }, 2000)
    this.#mocks.set(serverID, timer)
  }
}

export const metrics = new MetricsStore()
