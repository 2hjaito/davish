// Typed access to the Wails runtime. The frontend also runs in a plain browser
// (vite dev, previews), where window.runtime does not exist.
import {
  BrowserOpenURL,
  Quit,
  WindowIsMaximised,
  WindowMinimise,
  WindowSetBackgroundColour,
  WindowToggleMaximise,
} from '../../../wailsjs/runtime/runtime'

/**
 * The welcome page's "try it" mode: the app reloads with ?demo and runs on
 * the browser preview's sample data, connecting to nothing.
 */
export const isDemo =
  typeof location !== 'undefined' && new URLSearchParams(location.search).has('demo')

export const hasRuntime = typeof window !== 'undefined' && 'runtime' in window && !isDemo

/** lang carries the UI language over, since demo mode does not read settings.json. */
export function startDemo(lang: string) {
  location.search = `?demo&lang=${encodeURIComponent(lang)}`
}

export function exitDemo() {
  location.search = ''
}

/** Quits the app; a no-op outside Wails. */
export function quit() {
  if (hasRuntime) Quit()
}

/** Minimizes the app's OS window; a no-op outside Wails. */
export function minimizeWindow() {
  if (hasRuntime) WindowMinimise()
}

/** Maximizes the app's OS window, or restores it to its last size. */
export function toggleMaximizeWindow() {
  if (hasRuntime) WindowToggleMaximise()
}

/** Whether the app's OS window fills the screen; false outside Wails. */
export async function isWindowMaximized(): Promise<boolean> {
  return hasRuntime ? WindowIsMaximised() : false
}

/** Opens url in the user's default browser. */
export function openURL(url: string) {
  if (hasRuntime) BrowserOpenURL(url)
  else window.open(url, '_blank', 'noopener')
}

/**
 * Paints the native window behind the webview in the wallpaper's base color,
 * so a resize or a slow first frame does not flash the other theme.
 */
export function setWindowBackground(hex: string) {
  if (!hasRuntime || !/^#[0-9a-f]{6}$/i.test(hex)) return
  const [r, g, b] = [1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16))
  WindowSetBackgroundColour(r ?? 0, g ?? 0, b ?? 0, 255)
}
