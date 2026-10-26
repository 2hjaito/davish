// UI-only persistence until internal/store exposes settings.json (M2).
const PREFIX = 'davish:'

export function load<T>(key: string, fallback: T): T {
  try {
    const raw = localStorage.getItem(PREFIX + key)
    return raw === null ? fallback : { ...fallback, ...(JSON.parse(raw) as T) }
  } catch {
    return fallback
  }
}

export function save(key: string, value: unknown): void {
  try {
    localStorage.setItem(PREFIX + key, JSON.stringify(value))
  } catch {
    // Storage can be unavailable; the UI keeps working with in-memory state.
  }
}
