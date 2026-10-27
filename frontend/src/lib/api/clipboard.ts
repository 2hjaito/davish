// Clipboard access: the Wails runtime in the app, the browser API elsewhere.
import { ClipboardSetText } from '../../../wailsjs/runtime/runtime'
import { hasRuntime } from './runtime'

/** Copies text; false when the clipboard refused. */
export async function copyText(text: string): Promise<boolean> {
  try {
    if (hasRuntime) return await ClipboardSetText(text)
    await navigator.clipboard.writeText(text)
    return true
  } catch {
    return false
  }
}
