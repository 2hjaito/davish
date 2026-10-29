// UI state that survives restarts: window geometry, sort orders, toggles.
// In the app it lives in settings.json, written by Go (internal/store); in
// a plain browser (vite dev, previews) in localStorage. Call initPersist
// before anything calls load.
import { hasRuntime } from '../api/runtime'
import { settingsLoad, settingsSave } from '../api/settings'

const PREFIX = 'davish:'

/** settings.json content once loaded; null means localStorage is used. */
let cache: Map<string, unknown> | null = null
// Writes go out one at a time so an older value never lands after a newer one.
let writing: Promise<void> = Promise.resolve()

export async function initPersist(): Promise<void> {
  if (!hasRuntime) return
  let stored: Record<string, string> = {}
  try {
    stored = await settingsLoad()
  } catch {
    // Unreadable settings only cost preferences; start from defaults.
  }
  cache = new Map()
  for (const [key, json] of Object.entries(stored)) {
    try {
      cache.set(key, JSON.parse(json))
    } catch {
      // Skip a damaged key; its default applies.
    }
  }
  if (cache.size === 0) migrateLocalStorage()
}

// Before settings.json existed, everything lived in localStorage. Carry it
// over once so nobody loses their windows and toggles.
function migrateLocalStorage() {
  try {
    for (let i = 0; i < localStorage.length; i++) {
      const name = localStorage.key(i)
      if (!name?.startsWith(PREFIX)) continue
      const raw = localStorage.getItem(name)
      if (raw !== null) save(name.slice(PREFIX.length), JSON.parse(raw))
    }
  } catch {
    // Nothing to carry over.
  }
}

function read(key: string): unknown {
  if (cache) return cache.get(key)
  try {
    const raw = localStorage.getItem(PREFIX + key)
    return raw === null ? undefined : JSON.parse(raw)
  } catch {
    return undefined
  }
}

/** The stored value merged over fallback, so new fields get their defaults. */
export function load<T>(key: string, fallback: T): T {
  const stored = read(key)
  if (stored === undefined || stored === null) return fallback
  if (typeof fallback === 'object' && fallback !== null && typeof stored === 'object') {
    return { ...fallback, ...(stored as T) }
  }
  return stored as T
}

export function save(key: string, value: unknown): void {
  if (!cache) {
    try {
      localStorage.setItem(PREFIX + key, JSON.stringify(value))
    } catch {
      // Storage can be unavailable; the UI keeps working with in-memory state.
    }
    return
  }
  cache.set(key, value)
  const json = JSON.stringify(value)
  writing = writing
    .then(() => settingsSave(key, json))
    .catch(() => {
      // A failed write only loses a preference; the next change retries.
    })
}

/** Resolves once every save so far has reached settings.json. */
export function flushPersist(): Promise<void> {
  return writing
}
