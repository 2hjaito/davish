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

  it('lets a guard keep a window open until closed with force', () => {
    const a = wm.open('editor', 's1', opts)
    let asked = 0
    const off = wm.guardClose(a.id, () => {
      asked++
      return false
    })
    wm.close(a.id)
    expect(asked).toBe(1)
    expect(wm.windows).toHaveLength(1)
    wm.close(a.id, { force: true })
    expect(wm.windows).toHaveLength(0)
    off()
  })

  it('opens at the default size in the middle of the work area', () => {
    // Default area: x 0, y 30, 1280 × 670.
    expect(wm.open('terminal', 's1', opts).rect).toEqual({ x: 440, y: 215, w: 400, h: 300 })
  })

  it('cascades a window that would land exactly on another', () => {
    const a = wm.open('terminal', 's1', opts)
    const b = wm.open('files', 's1', opts)
    expect(b.rect).toEqual({ ...a.rect, x: a.rect.x + 28, y: a.rect.y + 28 })
    // Another workspace does not count.
    expect(wm.open('monitor', 's2', opts).rect).toEqual(a.rect)
  })

  it('pulls a saved window back fully on screen', () => {
    wm = new WindowManager({ files: { rect: { x: 1100, y: 600, w: 900, h: 2000 }, snap: null } })
    expect(wm.open('files', 's1', opts).rect).toEqual({ x: 380, y: 30, w: 900, h: 670 })
  })

  it('reopens maximized or snapped as it was left', () => {
    const a = wm.open('terminal', 's1', opts)
    wm.toggleMaximize(a.id)
    wm.close(a.id)
    expect(wm.open('terminal', 's1', opts).snap).toBe('top')
  })

  it('reads geometry saved by older versions', () => {
    wm = new WindowManager({ files: { x: 10, y: 40, w: 500, h: 400 } })
    const w = wm.open('files', 's1', opts)
    expect(w.rect).toEqual({ x: 10, y: 40, w: 500, h: 400 })
    expect(w.snap).toBeNull()
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

describe('layouts', () => {
  const options = (appId: string) =>
    appId === 'gone' ? undefined : { title: appId, size: { w: 400, h: 300 } }

  it('remembers what each server had open, back to front', () => {
    const wm = new WindowManager()
    const term = wm.open('terminal', 's1', opts)
    const files = wm.open('files', 's1', opts)
    wm.open('monitor', 's2', opts)
    wm.focus(term.id)
    wm.minimize(files.id)
    expect(wm.layoutOf('s1').map((l) => [l.appId, l.minimized])).toEqual([
      ['files', true],
      ['terminal', false],
    ])
    expect(wm.layoutOf('s2').map((l) => l.appId)).toEqual(['monitor'])
    wm.close(term.id)
    expect(wm.layoutOf('s1').map((l) => l.appId)).toEqual(['files'])
  })

  it('leaves out apps that need context to reopen', () => {
    const wm = new WindowManager()
    wm.keepInLayout((id) => id !== 'editor')
    wm.open('editor', 's1', opts)
    wm.open('files', 's1', opts)
    expect(wm.layoutOf('s1').map((l) => l.appId)).toEqual(['files'])
  })

  it('restores a layout once, into an empty workspace', () => {
    const rect = { x: 50, y: 60, w: 500, h: 400 }
    const wm = new WindowManager(
      {},
      {
        s1: [
          { appId: 'files', rect, snap: null, minimized: true },
          { appId: 'gone', rect, snap: null, minimized: false },
          { appId: 'terminal', rect, snap: 'left', minimized: false },
        ],
      },
    )
    expect(wm.restoreLayout('s1', options)).toBe(2)
    const [files, term] = wm.visible('s1')
    expect(files?.appId).toBe('files')
    expect(files?.minimized).toBe(true)
    expect(files?.rect).toEqual(rect)
    expect(term?.snap).toBe('left')
    expect(wm.focused('s1')?.appId).toBe('terminal')
    // Already populated: nothing more opens.
    expect(wm.restoreLayout('s1', options)).toBe(0)
  })

  it('opens an app where this server last had it', () => {
    const wm = new WindowManager()
    const a = wm.open('files', 's1', opts)
    wm.setRect(a.id, { x: 10, y: 40, w: 300, h: 200 })
    wm.commit(a.id)
    const b = wm.open('files', 's2', opts)
    wm.setRect(b.id, { x: 600, y: 40, w: 300, h: 200 })
    wm.commit(b.id)
    wm.close(a.id)
    wm.close(b.id)
    expect(wm.open('files', 's1', opts).rect.x).toBe(10)
    expect(wm.open('files', 's2', opts).rect.x).toBe(600)
    // A server that never had it: where the app was last used.
    expect(wm.open('files', 's3', opts).rect.x).toBe(600)
  })
})
