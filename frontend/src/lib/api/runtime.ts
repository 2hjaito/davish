// Typed access to the Wails runtime. The frontend also runs in a plain browser
// (vite dev, previews), where window.runtime does not exist.
import { Quit } from '../../../wailsjs/runtime/runtime'

export const hasRuntime = typeof window !== 'undefined' && 'runtime' in window

/** Quits the app; a no-op outside Wails. */
export function quit() {
  if (hasRuntime) Quit()
}
