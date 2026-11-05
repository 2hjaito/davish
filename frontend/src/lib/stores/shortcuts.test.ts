import { beforeEach, describe, expect, it, vi } from 'vitest'

import { shortcuts } from './shortcuts.svelte'

const S = 'server-1'
const PATH = '/srv/deploy.sh'
const PNG = 'data:image/png;base64,AAAA'
const grid = { cols: 4, rows: 4 }
const plain = { label: undefined, icon: undefined, color: undefined, image: '' }

const item = () => shortcuts.find(S, PATH)!
const stored = (id: string | undefined) =>
  localStorage.getItem(`davish:desktop-shortcut-image:${id}`)

const memory = new Map<string, string>()
vi.stubGlobal('localStorage', {
  getItem: (key: string) => memory.get(key) ?? null,
  setItem: (key: string, value: string) => memory.set(key, value),
})

beforeEach(() => {
  shortcuts.forgetServer(S)
  shortcuts.add(S, { path: PATH, name: 'deploy.sh', kind: 'script' }, { col: 0, row: 0 }, grid)
})

describe('customize', () => {
  it('keeps the name, path and trust of the file', () => {
    shortcuts.trust(S, PATH, 'aa')
    shortcuts.customize(S, PATH, { ...plain, label: 'Ship it', icon: 'zap', color: '#2f7de1' })
    expect(item()).toMatchObject({
      path: PATH,
      name: 'deploy.sh',
      trusted: 'aa',
      label: 'Ship it',
      icon: 'zap',
      color: '#2f7de1',
    })
  })

  it('takes a look back', () => {
    shortcuts.customize(S, PATH, { ...plain, label: 'Ship it', icon: 'zap' })
    shortcuts.customize(S, PATH, plain)
    expect(item().label).toBeUndefined()
    expect(item().icon).toBeUndefined()
  })

  it('keeps a picture out of the shortcut list, whose key is small', () => {
    shortcuts.customize(S, PATH, { ...plain, image: PNG })
    expect(shortcuts.image(item())).toBe(PNG)
    expect(JSON.stringify(shortcuts.byServer)).not.toContain(PNG)
  })

  it('keeps the same picture when only the name changes', () => {
    shortcuts.customize(S, PATH, { ...plain, image: PNG })
    const id = item().image
    shortcuts.customize(S, PATH, { ...plain, label: 'Ship it', image: PNG })
    expect(item().image).toBe(id)
  })

  it('gives a new picture a new id and frees the old one', () => {
    shortcuts.customize(S, PATH, { ...plain, image: PNG })
    const old = item().image
    shortcuts.customize(S, PATH, { ...plain, image: `${PNG}BBBB` })
    expect(item().image).not.toBe(old)
    expect(stored(old)).toBe('null')
  })

  it('removes a picture', () => {
    shortcuts.customize(S, PATH, { ...plain, image: PNG })
    shortcuts.customize(S, PATH, plain)
    expect(item().image).toBeUndefined()
    expect(shortcuts.image(item())).toBe('')
  })
})

describe('image', () => {
  it('shows nothing that is not a picture Davish encoded', () => {
    localStorage.setItem('davish:desktop-shortcut-image:x', '"https://example.com/a.png"')
    expect(shortcuts.image({ image: 'x' })).toBe('')
  })
})

describe('removing', () => {
  it('frees the picture with the shortcut', () => {
    shortcuts.customize(S, PATH, { ...plain, image: PNG })
    const id = item().image
    shortcuts.remove(S, PATH)
    expect(stored(id)).toBe('null')
  })

  it('frees the pictures of a forgotten server', () => {
    shortcuts.customize(S, PATH, { ...plain, image: PNG })
    const id = item().image
    shortcuts.forgetServer(S)
    expect(stored(id)).toBe('null')
  })
})
