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
})
