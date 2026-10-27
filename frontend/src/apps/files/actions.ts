// What can be done to a file or folder, from the drag menu or the keyboard.
import type { ArchiveFormat, Entry } from '../../lib/api/files'
import type { IconName } from '../../lib/ui/icons'

export type ActionId = 'editor' | 'terminal' | 'rename' | 'delete' | 'copyPath' | 'compress'

export interface ActionDef {
  id: ActionId
  icon: IconName
  /** Not built yet: shown, but cannot be chosen. */
  soon?: boolean
}

export const ACTIONS: Record<ActionId, ActionDef> = {
  editor: { id: 'editor', icon: 'editor', soon: true }, // the Editor arrives in M6
  terminal: { id: 'terminal', icon: 'terminal' },
  rename: { id: 'rename', icon: 'rename' },
  delete: { id: 'delete', icon: 'trash' },
  copyPath: { id: 'copyPath', icon: 'copy' },
  compress: { id: 'compress', icon: 'archive' },
}

/**
 * The four petals of the drag menu, clockwise from the top. Files and
 * folders get the actions that fit them; the rest stay on the keyboard.
 */
export function petalsFor(e: Pick<Entry, 'dir'>): ActionId[] {
  return e.dir
    ? ['terminal', 'rename', 'delete', 'compress']
    : ['editor', 'rename', 'delete', 'copyPath']
}

export const FORMATS: { id: ArchiveFormat; label: string }[] = [
  { id: 'zip', label: 'ZIP' },
  { id: 'tar.gz', label: 'GZ' },
  { id: 'tar', label: 'TAR' },
]

/** Keys that need no modifier, so they work on any keyboard layout. */
interface KeyInfo {
  key: string
  code: string
  primary: boolean
  shift: boolean
  alt: boolean
}

/**
 * The action a key press asks for. Option changes e.key on macOS, so
 * Option shortcuts match the physical key.
 */
export function actionForKey(k: KeyInfo): ActionId | null {
  if (!k.primary && !k.alt && !k.shift && k.key === 'F2') return 'rename'
  if (!k.primary && !k.alt && !k.shift && k.key === 'Delete') return 'delete'
  if (!k.primary || k.shift) return null
  if (k.alt) {
    if (k.code === 'KeyC') return 'copyPath'
    if (k.code === 'KeyZ') return 'compress'
    return null
  }
  if (k.key === 'Backspace') return 'delete'
  if (k.key.toLowerCase() === 't') return 'terminal'
  return null
}

/** Why newName cannot be used, or null when it can. */
export function renameProblem(oldName: string, newName: string): 'empty' | 'slash' | 'same' | null {
  const n = newName.trim()
  if (n === '' || n === '.' || n === '..') return 'empty'
  if (n.includes('/') || n.includes('\0')) return 'slash'
  if (n === oldName) return 'same'
  return null
}

/** The part of a name to preselect when renaming: all but the extension. */
export function stemLength(name: string): number {
  const dot = name.lastIndexOf('.')
  return dot > 0 ? dot : name.length
}
