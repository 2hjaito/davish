import * as api from '../api/lock'
import { onLockState } from '../api/events'
import { hasRuntime } from '../api/runtime'

/** Shortest lock password, as internal/lock.MinPasswordLen. */
export const MIN_LOCK_PASSWORD = 6

/** Idle minutes before the app locks itself; 0 is never. */
export const LOCK_AFTER = [0, 5, 15, 30, 60] as const

/**
 * The app lock (ADR 0007). Go holds the truth; this mirrors lock:state.
 * In the plain browser preview a password kept in memory stands in.
 */
class AppLock {
  state = $state<api.LockState>({ enabled: false, locked: false, biometric: false, retryAt: 0 })
  ready = $state(false)
  /** The desktop mounts on the first unlock; later locks only cover it. */
  everUnlocked = $state(false)
  #demoPassword = ''

  async init(): Promise<() => void> {
    if (!hasRuntime) {
      this.#set({ ...this.state })
      this.ready = true
      return () => {}
    }
    const off = onLockState((s) => this.#set(s))
    this.#set(await api.lockState())
    this.ready = true
    return off
  }

  #set(s: api.LockState) {
    this.state = s
    if (!s.locked) this.everUnlocked = true
  }

  async #refresh() {
    if (hasRuntime) this.#set(await api.lockState())
  }

  lock() {
    if (!this.state.enabled || this.state.locked) return
    if (hasRuntime) void api.lockNow()
    else this.#set({ ...this.state, locked: true })
  }

  /** Rejects with LOCK_WRONG_PASSWORD or LOCK_TOO_MANY_ATTEMPTS. */
  async unlock(password: string) {
    if (!hasRuntime) {
      if (password !== this.#demoPassword) throw { code: 'LOCK_WRONG_PASSWORD' }
      this.#set({ ...this.state, locked: false })
      return
    }
    try {
      await api.unlock(password)
    } finally {
      // A wrong password may have started a wait.
      await this.#refresh()
    }
  }

  async touchID(reason: string) {
    if (hasRuntime) await api.unlockTouchID(reason)
    await this.#refresh()
  }

  async setPassword(current: string, next: string) {
    if (!hasRuntime) {
      if (this.state.enabled && current !== this.#demoPassword)
        throw { code: 'LOCK_WRONG_PASSWORD' }
      this.#demoPassword = next
      this.#set({ ...this.state, enabled: true })
      return
    }
    await api.lockSetPassword(current, next)
    await this.#refresh()
  }

  async disable(current: string) {
    if (!hasRuntime) {
      if (current !== this.#demoPassword) throw { code: 'LOCK_WRONG_PASSWORD' }
      this.#set({ ...this.state, enabled: false })
      return
    }
    await api.lockDisable(current)
    await this.#refresh()
  }

  async reset() {
    if (hasRuntime) await api.lockReset()
    this.#demoPassword = ''
    this.#set({ enabled: false, locked: false, biometric: false, retryAt: 0 })
  }
}

export const appLock = new AppLock()
