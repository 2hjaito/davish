import { beforeEach, describe, expect, it, vi } from 'vitest'

let stored: unknown = null
const saves: [string, unknown][] = []
const relaunch = vi.fn<() => Promise<void>>()

vi.mock('../api/runtime', () => ({ hasRuntime: true }))
vi.mock('../api/update', () => ({ relaunch: () => relaunch() }))
vi.mock('../stores/persist', () => ({
  load: (_key: string, fallback: unknown) => stored ?? fallback,
  save: (key: string, value: unknown) => saves.push([key, value]),
  flushPersist: () => Promise.resolve(),
}))

const reload = vi.fn()
const replace = vi.fn()

function system(language: string, search = '') {
  vi.stubGlobal('navigator', { language, userAgent: '' })
  vi.stubGlobal('location', { search, href: `app://davish/${search}`, reload, replace })
}

async function fresh() {
  vi.resetModules()
  return import('.')
}

describe('i18n', () => {
  beforeEach(() => {
    stored = null
    saves.length = 0
    relaunch.mockReset()
    reload.mockReset()
    replace.mockReset()
    system('vi-VN')
  })

  it('follows the system language until one is chosen', async () => {
    const { savedLang } = await fresh()
    expect(savedLang()).toBe('vi')
    system('en-GB')
    expect(savedLang()).toBe('en')
    system('fr-FR')
    expect(savedLang()).toBe('en')
  })

  it('prefers the saved choice, and the URL over both', async () => {
    const { savedLang } = await fresh()
    stored = 'en'
    expect(savedLang()).toBe('en')
    system('en-US', '?demo&lang=vi')
    expect(savedLang()).toBe('vi')
    stored = 'klingon'
    system('vi-VN')
    expect(savedLang()).toBe('vi')
  })

  it('loads the strings and locale of a language', async () => {
    const i18n = await fresh()
    await i18n.loadLang('en')
    expect(i18n.lang).toBe('en')
    expect(i18n.locale).toBe('en-US')
    expect(i18n.t.apps.settings).toBe('Settings')
    await i18n.loadLang('vi')
    expect(i18n.t.apps.settings).toBe('Cài đặt')
  })

  it('saves the new language and restarts Davish', async () => {
    const i18n = await fresh()
    await i18n.loadLang('vi')
    relaunch.mockResolvedValue()
    await i18n.switchLang('en')
    expect(saves).toEqual([['lang', 'en']])
    expect(relaunch).toHaveBeenCalledOnce()
    expect(reload).not.toHaveBeenCalled()
  })

  it('reloads the page when Davish cannot restart itself', async () => {
    const i18n = await fresh()
    await i18n.loadLang('vi')
    relaunch.mockRejectedValue(new Error('not an app bundle'))
    await i18n.switchLang('en')
    expect(reload).toHaveBeenCalledOnce()
  })

  it('does nothing for the current language', async () => {
    const i18n = await fresh()
    await i18n.loadLang('en')
    await i18n.switchLang('en')
    expect(saves).toEqual([])
    expect(relaunch).not.toHaveBeenCalled()
  })
})
