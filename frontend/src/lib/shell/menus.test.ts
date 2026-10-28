import { describe, expect, it } from 'vitest'

import { appMenus, findShortcut, matches, type Menu } from './menus.svelte'

const key = (k: string, shiftKey = false, code = '') => ({ key: k, shiftKey, code })

describe('matches', () => {
  it('matches letters by physical key, with Shift as written', () => {
    expect(matches('K', key('k', false, 'KeyK'))).toBe(true)
    expect(matches('K', key('K', true, 'KeyK'))).toBe(false)
    expect(matches('shift+A', key('A', true, 'KeyA'))).toBe(true)
    expect(matches('shift+A', key('a', false, 'KeyA'))).toBe(false)
    // A non-Latin layout still reports the physical key.
    expect(matches('W', key('ц', false, 'KeyW'))).toBe(true)
  })

  it('matches punctuation by character whatever Shift does', () => {
    expect(matches('/', key('/', true, 'Digit7'))).toBe(true)
    expect(matches(',', key(',', false, 'Comma'))).toBe(true)
  })
})

describe('findShortcut', () => {
  const ran: string[] = []
  const menus: Menu[] = [
    {
      id: 'a',
      label: 'A',
      items: [
        { label: 'off', combo: 'W', disabled: true, run: () => ran.push('off') },
        null,
        { label: 'close', combo: 'W', run: () => ran.push('close') },
      ],
    },
  ]

  it('skips disabled items and separators', () => {
    findShortcut(menus, key('w', false, 'KeyW'))?.run()
    expect(ran).toEqual(['close'])
    expect(findShortcut(menus, key('q', false, 'KeyQ'))).toBeUndefined()
  })
})

describe('appMenus', () => {
  it('joins what a window added with separators and forgets it on removal', () => {
    const r = () => {}
    const off1 = appMenus.add('w1', { items: () => [{ label: 'a', run: r }], refresh: r })
    const off2 = appMenus.add('w1', { items: () => [{ label: 'b', run: r }] })
    expect(appMenus.items('w1').map((x) => x?.label ?? '-')).toEqual(['a', '-', 'b'])
    expect(appMenus.refresh('w1')).toBe(r)
    off1()
    off2()
    expect(appMenus.items('w1')).toEqual([])
    expect(appMenus.refresh('w1')).toBeUndefined()
  })
})
