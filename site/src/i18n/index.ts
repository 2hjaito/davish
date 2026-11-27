import { en, type Dict } from './en'
import { vi } from './vi'
import { zh } from './zh'
import { ko } from './ko'
import { fr } from './fr'

export type Locale = 'en' | 'vi' | 'zh' | 'ko' | 'fr'

export const dicts: Record<Locale, Dict> = { en, vi, zh, ko, fr }

/** Each locale's path and the name it goes by in its own language. */
export const locales: { id: Locale; path: string; name: string; hreflang: string }[] = [
  { id: 'en', path: '/', name: 'English', hreflang: 'en' },
  { id: 'vi', path: '/vi/', name: 'Tiếng Việt', hreflang: 'vi' },
  { id: 'zh', path: '/zh/', name: '简体中文', hreflang: 'zh-CN' },
  { id: 'ko', path: '/ko/', name: '한국어', hreflang: 'ko' },
  { id: 'fr', path: '/fr/', name: 'Français', hreflang: 'fr' },
]
