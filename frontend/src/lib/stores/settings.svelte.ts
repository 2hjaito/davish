import { load, save } from './persist'

export const WALLPAPER_ACCENTS = ['#8fb2ff', '#c3a6ff', '#9fe0c8', '#ffb38a'] as const

interface SettingsData {
  reduceGlass: boolean
  reduceMotion: boolean
  wallpaperAccent: string
}

const defaults: SettingsData = {
  reduceGlass: false,
  reduceMotion: false,
  wallpaperAccent: WALLPAPER_ACCENTS[0],
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

  set<K extends keyof SettingsData>(key: K, value: SettingsData[K]) {
    this.#data[key] = value
    save('settings', this.#data)
  }
}

export const settings = new Settings()
