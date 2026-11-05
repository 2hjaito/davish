// Typed wrappers for the desktop shortcut bindings (internal/app/shortcuts.go).
import * as App from '../../../wailsjs/go/app/App'

import { hasRuntime } from './runtime'

export type ShortcutKind = 'folder' | 'script' | 'file'

/** What a path is right now (internal/shortcut.Target). */
export interface ShortcutTarget {
  name: string
  kind: ShortcutKind
  /** SHA-256 of a script; empty when it is too big to read. */
  hash: string
  /** The command a script's run does, shown before it runs. */
  command: string
}

export interface ShortcutRun {
  line: string
  nonce: string
}

// The browser preview has no server: names decide, as a stand-in.
function mockTarget(path: string): ShortcutTarget {
  const name = path.split('/').pop() ?? path
  const dir = path.slice(0, -name.length - 1) || '/'
  if (name.endsWith('.sh')) {
    return { name, kind: 'script', hash: 'preview', command: `cd -- ${dir} && ./${name}` }
  }
  return { name, kind: name.includes('.') ? 'file' : 'folder', hash: '', command: '' }
}

export const shortcutInspect = (serverID: string, path: string): Promise<ShortcutTarget> =>
  hasRuntime
    ? (App.ShortcutInspect(serverID, path) as Promise<ShortcutTarget>)
    : Promise.resolve(mockTarget(path))

export const shortcutRun = (serverID: string, path: string, hash: string): Promise<ShortcutRun> =>
  hasRuntime
    ? App.ShortcutRun(serverID, path, hash)
    : Promise.resolve({ line: '', nonce: Date.now().toString(16) })
