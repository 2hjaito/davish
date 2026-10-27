import { load, save } from './persist'

export const WALLPAPER_ACCENTS = ['#8fb2ff', '#c3a6ff', '#9fe0c8', '#ffb38a'] as const

export const TERMINAL_FONT = { min: 10, max: 20, step: 0.5, default: 12.5 } as const

/** Keeps a terminal font size in range and on the step grid; junk falls back to the default. */
export function clampFontSize(size: number): number {
  if (!Number.isFinite(size)) return TERMINAL_FONT.default
  const snapped = Math.round(size / TERMINAL_FONT.step) * TERMINAL_FONT.step
  return Math.min(TERMINAL_FONT.max, Math.max(TERMINAL_FONT.min, snapped))
}

interface SettingsData {
  reduceGlass: boolean
  reduceMotion: boolean
  wallpaperAccent: string
  terminalFontSize: number
  terminalGpu: boolean
}

const defaults: SettingsData = {
  reduceGlass: false,
  reduceMotion: false,
  wallpaperAccent: WALLPAPER_ACCENTS[0],
  terminalFontSize: TERMINAL_FONT.default,
  terminalGpu: false,
}

class Settings {
  #data = $state<SettingsData>(load('settings', defaults))

  get reduceGlass() {
    return this.#data.reduceGlass
  }
  get reduceMotion() {
    return this.#data.reduceMotion
  }
  get wallpaperAccent() {
    return this.#data.wallpaperAccent
  }
  get terminalGpu() {
    return this.#data.terminalGpu === true
  }
  get terminalFontSize() {
    return clampFontSize(this.#data.terminalFontSize)
  }

  set<K extends keyof SettingsData>(key: K, value: SettingsData[K]) {
    this.#data[key] = value
    save('settings', this.#data)
  }
}

export const settings = new Settings()
