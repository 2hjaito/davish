import { beforeEach, describe, expect, it, vi } from 'vitest'

const saved = new Map<string, string>()
let stored: Record<string, string> = {}

vi.mock('../api/runtime', () => ({ hasRuntime: true }))
vi.mock('../api/settings', () => ({
  settingsLoad: () => Promise.resolve(stored),
  settingsSave: (key: string, json: string) => {
    saved.set(key, json)
    return Promise.resolve()
  },
}))

// Tests run in node, which has no localStorage.
class MemoryStorage {
  #items = new Map<string, string>()
  get length() {
    return this.#items.size
  }
  key(i: number) {
    return [...this.#items.keys()][i] ?? null
  }
  getItem(k: string) {
    return this.#items.get(k) ?? null
  }
  setItem(k: string, v: string) {
    this.#items.set(k, v)
  }
  clear() {
    this.#items.clear()
  }
}
vi.stubGlobal('localStorage', new MemoryStorage())

// A fresh module per test: initPersist fills module state once.
async function fresh() {
  vi.resetModules()
  const m = await import('./persist')
  await m.initPersist()
  return m
}

const flush = () => new Promise((r) => setTimeout(r, 0))

describe('persist with settings.json', () => {
  beforeEach(() => {
    saved.clear()
    stored = {}
    localStorage.clear()
  })

  it('reads stored keys, merging objects over their defaults', async () => {
    stored = { files: '{"order":{"key":"size","desc":true}}', broken: '{nope' }
    const { load } = await fresh()
    expect(load('files', { order: { key: 'name', desc: false }, extra: 1 })).toEqual({
      order: { key: 'size', desc: true },
      extra: 1,
    })
    expect(load('broken', { a: 1 })).toEqual({ a: 1 })
    expect(load('missing', 5)).toBe(5)
  })

  it('keeps stored arrays as arrays and drops a mismatched shape', async () => {
    stored = { 'dock:pinned': '["files","terminal"]', 'logs:units': '{"a":1}' }
    const { load } = await fresh()
    expect(load('dock:pinned', ['terminal', 'files', 'monitor'])).toEqual(['files', 'terminal'])
    expect(load('logs:units', ['x'])).toEqual(['x'])
  })

  it('writes through to Go and serves the new value at once', async () => {
    const { load, save } = await fresh()
    save('monitor', { range: '15m' })
    expect(load('monitor', { range: '5m' })).toEqual({ range: '15m' })
    await flush()
    expect(saved.get('monitor')).toBe('{"range":"15m"}')
  })

  it('carries localStorage over when settings.json is empty', async () => {
    localStorage.setItem('davish:settings', '{"reduceGlass":true}')
    localStorage.setItem('other-app', '1')
    const { load } = await fresh()
    expect(load('settings', { reduceGlass: false })).toEqual({ reduceGlass: true })
    await flush()
    expect([...saved.keys()]).toEqual(['settings'])
  })

  it('does not migrate over existing settings', async () => {
    stored = { settings: '{"reduceGlass":false}' }
    localStorage.setItem('davish:settings', '{"reduceGlass":true}')
    const { load } = await fresh()
    expect(load('settings', { reduceGlass: true })).toEqual({ reduceGlass: false })
    expect(saved.size).toBe(0)
  })
})
