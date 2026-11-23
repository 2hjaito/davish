import { describe, expect, it } from 'vitest'

import { settings } from '../stores/settings.svelte'
import { ui } from '../stores/ui.svelte'
import { WIDGET_KINDS } from '../widgets/kinds'
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

describe('widgets in the View menu', () => {
  const view = () =>
    buildMenus()
      .find((m) => m.id === 'view')!
      .items.filter((it) => it !== null)

  it('has an edit switch and one add item per kind of widget', () => {
    settings.usePreset('beautiful')
    ui.widgetEdit = false
    const labels = view().map((it) => it.label)
    expect(labels).toContain('Chỉnh sửa widget')
    expect(labels.filter((l) => l.startsWith('Thêm widget')).length).toBe(WIDGET_KINDS.length)
  })

  it('says Done while editing', () => {
    settings.usePreset('beautiful')
    ui.widgetEdit = true
    expect(view().map((it) => it.label)).toContain('Xong chỉnh sửa widget')
    ui.widgetEdit = false
  })

  it('is disabled when widgets are off in the performance preset', () => {
    settings.usePreset('light')
    const items = view().filter((it) => /widget/i.test(it.label))
    expect(items.length).toBeGreaterThan(1)
    expect(items.every((it) => it.disabled)).toBe(true)
    settings.usePreset('beautiful')
  })
})
