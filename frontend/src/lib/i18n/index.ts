// The UI language, picked once before the app mounts (ADR 0010). Only the
// chosen strings are loaded; switching saves the choice and restarts Davish.
import { hasRuntime } from '../api/runtime'
import { relaunch } from '../api/update'
import type { IconName } from '../ui/icons'
import { flushPersist, load, save } from '../stores/persist'
import type vi from './vi'

export type { AgoUnit } from './vi'

export type Lang = 'vi' | 'en'

export const LANGS: readonly Lang[] = ['vi', 'en']

// vi.ts is `as const`; another language must match its shape, not its text.
// Icon names stay literal since they are not translated.
type Widen<T> = T extends IconName
  ? T
  : T extends string
    ? string
    : T extends (...args: infer A) => infer R
      ? (...args: A) => Widen<R>
      : T extends readonly (infer U)[]
        ? T extends readonly [unknown, ...unknown[]]
          ? { readonly [K in keyof T]: Widen<T[K]> }
          : readonly Widen<U>[]
        : T extends object
          ? { readonly [K in keyof T]: Widen<T[K]> }
          : T

export type Messages = Widen<typeof vi>

const LOCALES: Record<Lang, string> = { vi: 'vi-VN', en: 'en-US' }

/** Every user-facing string, in the current language. */
export let t: Messages
/** The current language. */
export let lang: Lang
/** For Intl and toLocale*: numbers and dates follow the UI language. */
export let locale: string

function isLang(v: unknown): v is Lang {
  return v === 'vi' || v === 'en'
}

/** The saved choice; before one is made, the system's language. */
export function savedLang(): Lang {
  // Demo mode keeps its state apart from settings.json, so it is handed the
  // language in the URL (startDemo); previews and screenshots use it too.
  const param = typeof location === 'undefined' ? null : urlLang()
  if (isLang(param)) return param
  const saved = load<unknown>('lang', null)
  if (isLang(saved)) return saved
  const sys = typeof navigator === 'undefined' ? 'vi' : navigator.language
  return sys.toLowerCase().startsWith('vi') ? 'vi' : 'en'
}

/** Loads the strings of pick (the saved language by default); call before mounting. */
export async function loadLang(pick: Lang = savedLang()): Promise<void> {
  const mod = pick === 'en' ? await import('./en') : await import('./vi')
  t = mod.default
  lang = pick
  locale = LOCALES[pick]
}

/**
 * Saves next and starts Davish again in it. A restart rather than a page
 * reload, so no terminal or log stream of the old page lives on in Go.
 */
export async function switchLang(next: Lang): Promise<void> {
  if (next === lang) return
  save('lang', next)
  await flushPersist()
  if (hasRuntime) {
    try {
      await relaunch()
      return
    } catch {
      // wails dev runs outside an app bundle: reloading the page will do.
    }
  }
  if (urlLang() === null) {
    location.reload()
    return
  }
  const url = new URL(location.href)
  url.searchParams.set('lang', next)
  location.replace(url)
}

function urlLang(): string | null {
  return new URLSearchParams(location.search).get('lang')
}
