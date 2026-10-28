// The menu bar's model: entries, their shortcuts, and the commands each
// open app adds for its focused window.
import { onDestroy } from 'svelte'

import { getWindowContext } from '../wm/context'
import { isMac } from './platform'

/**
 * A shortcut with the platform's primary modifier (⌘ on macOS, Ctrl
 * elsewhere): a key, optionally with Shift, such as 'K' or 'shift+A'.
 */
export type Combo = string

export interface MenuItem {
  label: string
  run: () => void
  combo?: Combo
  disabled?: boolean
  checked?: boolean
}

/** null is a separator. */
export type MenuEntry = MenuItem | null

export interface Menu {
  id: string
  label: string
  bold?: boolean
  items: MenuEntry[]
}

function parse(combo: Combo): { key: string; shift: boolean } {
  const shift = combo.startsWith('shift+')
  return { key: (shift ? combo.slice(6) : combo).toLowerCase(), shift }
}

/** The label for a combo, such as ⌘⇧A or Ctrl+Shift+A. */
export function comboLabel(combo: Combo): string {
  const { key, shift } = parse(combo)
  const k = key.length === 1 ? key.toUpperCase() : key
  if (isMac) return `⌘${shift ? '⇧' : ''}${k}`
  return `Ctrl+${shift ? 'Shift+' : ''}${k}`
}

/** True when e (with the primary modifier already held) is combo. */
export function matches(combo: Combo, e: Pick<KeyboardEvent, 'key' | 'shiftKey' | 'code'>) {
  const { key, shift } = parse(combo)
  // '/' and ',' sit under Shift on some layouts; match them by character.
  if (key.length === 1 && !/[a-z0-9]/.test(key)) return e.key === key
  const pressed = /^Key[A-Z]$/.test(e.code) ? e.code.slice(3).toLowerCase() : e.key.toLowerCase()
  return pressed === key && e.shiftKey === shift
}

/** The first enabled item in menus bound to the pressed combo. */
export function findShortcut(
  menus: readonly Menu[],
  e: Pick<KeyboardEvent, 'key' | 'shiftKey' | 'code'>,
): MenuItem | undefined {
  for (const m of menus)
    for (const it of m.items) if (it?.combo && !it.disabled && matches(it.combo, e)) return it
  return undefined
}

/** What an app adds for one window: its own menu and, optionally, refresh. */
interface Contribution {
  items: () => MenuEntry[]
  refresh?: () => void
}

class AppMenus {
  #byWindow = $state<Record<string, Contribution[]>>({})

  add(winId: string, c: Contribution): () => void {
    this.#byWindow[winId] = [...(this.#byWindow[winId] ?? []), c]
    return () => {
      const rest = (this.#byWindow[winId] ?? []).filter((x) => x !== c)
      if (rest.length) this.#byWindow[winId] = rest
      else delete this.#byWindow[winId]
    }
  }

  items(winId: string): MenuEntry[] {
    const out: MenuEntry[] = []
    for (const c of this.#byWindow[winId] ?? []) {
      const items = c.items()
      if (items.length && out.length) out.push(null)
      out.push(...items)
    }
    return out
  }

  refresh(winId: string): (() => void) | undefined {
    return (this.#byWindow[winId] ?? []).find((c) => c.refresh)?.refresh
  }
}

export const appMenus = new AppMenus()

/**
 * Adds commands to the menu bar while this component's window is focused.
 * Call it during component setup, inside a window.
 */
export function useAppMenu(items: () => MenuEntry[], refresh?: () => void) {
  const { win } = getWindowContext()
  onDestroy(appMenus.add(win.id, { items, refresh }))
}
