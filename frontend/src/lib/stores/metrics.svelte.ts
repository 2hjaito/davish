import { onSysmonTick, type SysmonTick, type Unsubscribe } from '../api/events'
import { monitorUnwatch, monitorWatch } from '../api/monitor'
import { hasRuntime } from '../api/runtime'

export type Metrics = Omit<SysmonTick, 'serverID'>

const GB = 1024 ** 3

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

  // A plain browser (vite dev, previews) has no Go side: jitter demo numbers.
  #startMock(serverID: string) {
    this.byServer[serverID] = {
      cpu: 23,
      memUsed: 5.1 * GB,
      memTotal: 16 * GB,
      diskPct: 62,
      rxBps: 1.2 * 1024 ** 2,
      txBps: 0.2 * 1024 ** 2,
      load1: 0.4,
      uptime: 86400,
    }
    const jitter = (v: number, d: number, min: number, max: number) =>
      Math.min(max, Math.max(min, v + (Math.random() * 2 - 1) * d))
    const timer = setInterval(() => {
      const c = this.byServer[serverID]
      if (!c) return
      this.byServer[serverID] = {
        ...c,
        cpu: Math.round(jitter(c.cpu, 6, 3, 97)),
        memUsed: jitter(c.memUsed, 0.15 * GB, 3 * GB, 9 * GB),
        rxBps: jitter(c.rxBps, 300 * 1024, 20 * 1024, 4 * 1024 ** 2),
        txBps: jitter(c.txBps, 80 * 1024, 5 * 1024, 1024 ** 2),
        uptime: c.uptime + 2,
      }
    }, 2000)
    this.#mocks.set(serverID, timer)
  }
}

export const metrics = new MetricsStore()
