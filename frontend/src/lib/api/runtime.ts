// Typed access to the Wails runtime. The frontend also runs in a plain browser
// (vite dev, previews), where window.runtime does not exist.
import { BrowserOpenURL, Quit } from '../../../wailsjs/runtime/runtime'

/**
 * The welcome page's "try it" mode: the app reloads with ?demo and runs on
 * the browser preview's sample data, connecting to nothing.
 */
export const isDemo =
  typeof location !== 'undefined' && new URLSearchParams(location.search).has('demo')

export const hasRuntime = typeof window !== 'undefined' && 'runtime' in window && !isDemo

export function startDemo() {
  location.search = '?demo'
}

export function exitDemo() {
  location.search = ''
}

/** Quits the app; a no-op outside Wails. */
export function quit() {
  if (hasRuntime) Quit()
}

/** Opens url in the user's default browser. */
export function openURL(url: string) {
  if (hasRuntime) BrowserOpenURL(url)
  else window.open(url, '_blank', 'noopener')
}
