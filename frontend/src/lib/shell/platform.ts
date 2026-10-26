export const isMac = typeof navigator !== 'undefined' && navigator.userAgent.includes('Mac')

/** Label for a shortcut with the platform's primary modifier. */
export function shortcut(key: string): string {
  return isMac ? `⌘${key}` : `Ctrl+${key}`
}

export function hasPrimaryModifier(e: KeyboardEvent): boolean {
  return isMac ? e.metaKey && !e.ctrlKey : e.ctrlKey && !e.metaKey
}

/** True inside the Wails app on macOS, where the OS draws traffic lights over the menu bar. */
export const hasNativeTrafficLights = isMac && typeof window !== 'undefined' && 'runtime' in window
