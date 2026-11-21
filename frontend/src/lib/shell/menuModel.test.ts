import { describe, expect, it } from 'vitest'

import { buildMenus } from './menuModel.svelte'

describe('buildMenus', () => {
  // #202: All servers and Services both took ⌘⇧S, and only one of them ran.
  it('gives every shortcut to one menu item only', () => {
    const seen = new Map<string, string>()
    const clashes: string[] = []
    for (const m of buildMenus())
      for (const it of m.items) {
        if (!it?.combo) continue
        const prev = seen.get(it.combo)
        if (prev) clashes.push(`${it.combo}: ${prev} / ${it.label}`)
        else seen.set(it.combo, it.label)
      }
    expect(clashes).toEqual([])
  })

  // #209: the Windows webview keeps Ctrl+0, Ctrl+= and Ctrl+- for zoom
  // and never passes them on, so a menu item there would never run.
  it('leaves the zoom keys to the webview', () => {
    const zoom = ['0', '=', '-', '+']
    const taken = buildMenus()
      .flatMap((m) => m.items)
      .filter((it) => it?.combo && zoom.includes(it.combo.replace('shift+', '')))
      .map((it) => it?.label)
    expect(taken).toEqual([])
  })

  // The primary modifier is ⌘ on macOS and Ctrl elsewhere. These are the
  // keys the OS or the webview takes before the page sees them (see
  // AGENT_REVIEW.md 2.2); a menu item on one would silently never run.
  const reserved: Record<string, { plain: string[]; any?: string[] }> = {
    macOS: { plain: ['Q', 'H', 'Tab', ' ', '`'] },
    Windows: { plain: ['P'], any: ['0', '=', '-', '+'] },
    Linux: { plain: [], any: ['0', '=', '-', '+'] },
  }
  // Wails adds no Window menu on macOS (main.go builds only AppMenu and
  // EditMenu), so ⌘M is free for Minimize there. Not tried by hand yet.
  const allowed = new Set(['M'])

  for (const [os, keys] of Object.entries(reserved)) {
    it(`keeps menu shortcuts off the keys ${os} reserves`, () => {
      const taken = buildMenus()
        .flatMap((m) => m.items)
        .filter((it) => it?.combo)
        .filter((it) => {
          const combo = it?.combo ?? ''
          const shifted = combo.startsWith('shift+')
          const key = combo.replace('shift+', '')
          if (allowed.has(key) && os === 'macOS') return false
          return keys.any?.includes(key) || (!shifted && keys.plain.includes(key.toUpperCase()))
        })
        .map((it) => `${it?.label} (${it?.combo})`)
      expect(taken).toEqual([])
    })
  }
})
