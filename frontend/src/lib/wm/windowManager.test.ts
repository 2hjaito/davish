import { beforeEach, describe, expect, it } from 'vitest'

import { WindowManager } from './windowManager.svelte'

const opts = { title: 'T', size: { w: 400, h: 300 } }

describe('WindowManager', () => {
  let wm: WindowManager

  beforeEach(() => {
    wm = new WindowManager()
  })

  it('focuses the newest window', () => {
    const a = wm.open('terminal', 's1', opts)
    const b = wm.open('files', 's1', opts)
    expect(wm.focused('s1')?.id).toBe(b.id)
    wm.focus(a.id)
    expect(wm.focused('s1')?.id).toBe(a.id)
  })

  it('keeps workspaces separate', () => {
    wm.open('terminal', 's1', opts)
    wm.open('files', 's2', opts)
    expect(wm.visible('s1').map((w) => w.appId)).toEqual(['terminal'])
    expect(wm.isOpen('files', 's1')).toBe(false)
  })

  it('reuses an existing window instead of opening a second one', () => {
    expect(wm.openOrFocus('terminal', 's1', opts)).toBe(true)
    expect(wm.openOrFocus('terminal', 's1', opts)).toBe(false)
    expect(wm.windows).toHaveLength(1)
  })

  it('skips minimized windows when finding focus and restores on focus', () => {
    const a = wm.open('terminal', 's1', opts)
    const b = wm.open('files', 's1', opts)
    wm.minimize(b.id)
    expect(wm.focused('s1')?.id).toBe(a.id)
    wm.focus(b.id)
    expect(wm.windows.find((w) => w.id === b.id)?.minimized).toBe(false)
  })

  it('maximizes to the work area and back', () => {
    const a = wm.open('terminal', 's1', opts)
    wm.toggleMaximize(a.id)
    expect(wm.frame(wm.windows[0]!)).toEqual(wm.area)
    wm.toggleMaximize(a.id)
    expect(wm.frame(wm.windows[0]!)).toEqual(a.rect)
  })

  it('reopens an app where it was closed', () => {
    const a = wm.open('terminal', 's1', opts)
    wm.setRect(a.id, { x: 200, y: 150, w: 500, h: 350 })
    wm.close(a.id)
    expect(wm.open('terminal', 's1', opts).rect).toEqual({ x: 200, y: 150, w: 500, h: 350 })
  })

  it('keeps the pointer position when dragging out of a snap', () => {
    const a = wm.open('terminal', 's1', opts)
    wm.snap(a.id, 'left')
    const half = wm.frame(wm.windows[0]!)
    const px = half.x + half.w / 2
    const r = wm.unsnapForDrag(a.id, px)!
    expect(r.x + r.w / 2).toBeCloseTo(px, 0)
    expect(wm.windows[0]!.snap).toBeNull()
  })
})
