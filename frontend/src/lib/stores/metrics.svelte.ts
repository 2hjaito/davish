export interface Metrics {
  cpu: number
  memUsed: number
  memTotal: number
  diskPct: number
  rxBps: number
  txBps: number
}

const GB = 1024 ** 3

// Fake ticks until sysmon lands in M4; the shape matches sysmon:tick.
class MetricsStore {
  current = $state<Metrics>({
    cpu: 23,
    memUsed: 5.1 * GB,
    memTotal: 16 * GB,
    diskPct: 62,
    rxBps: 1.2 * 1024 ** 2,
    txBps: 0.2 * 1024 ** 2,
  })

  #timer: ReturnType<typeof setInterval> | undefined
  #subscribers = 0

  /** Starts ticking while at least one subscriber needs it. */
  subscribe(): () => void {
    if (this.#subscribers++ === 0) {
      this.#timer = setInterval(() => this.#tick(), 2000)
    }
    return () => {
      if (--this.#subscribers === 0) clearInterval(this.#timer)
    }
  }

  #tick() {
    const c = this.current
    const jitter = (v: number, d: number, min: number, max: number) =>
      Math.min(max, Math.max(min, v + (Math.random() * 2 - 1) * d))
    this.current = {
      ...c,
      cpu: Math.round(jitter(c.cpu, 6, 3, 97)),
      memUsed: jitter(c.memUsed, 0.15 * GB, 3 * GB, 9 * GB),
      rxBps: jitter(c.rxBps, 300 * 1024, 20 * 1024, 4 * 1024 ** 2),
      txBps: jitter(c.txBps, 80 * 1024, 5 * 1024, 1024 ** 2),
    }
  }
}

export const metrics = new MetricsStore()
