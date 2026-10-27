import { load, save } from './persist'

export const WALLPAPER_ACCENTS = ['#8fb2ff', '#c3a6ff', '#9fe0c8', '#ffb38a'] as const

export const TERMINAL_FONT = { min: 10, max: 20, step: 0.5, default: 12.5 } as const

/** Keeps a terminal font size in range and on the step grid; junk falls back to the default. */
export function clampFontSize(size: number): number {
  if (!Number.isFinite(size)) return TERMINAL_FONT.default
  const snapped = Math.round(size / TERMINAL_FONT.step) * TERMINAL_FONT.step
  return Math.min(TERMINAL_FONT.max, Math.max(TERMINAL_FONT.min, snapped))
}

/** The monospace font of Terminal, Editor and Logs. */
export type MonoFont = 'jetbrains' | 'system'

export const MONO_FAMILIES: Record<MonoFont, string> = {
  // Bundled with the app; the Nerd Font symbols back the Davish prompt.
  jetbrains:
    "'JetBrains Mono Variable', 'JetBrains Mono', 'Symbols Nerd Font Mono', ui-monospace, monospace",
  system:
    "ui-monospace, 'SF Mono', Menlo, Consolas, 'Cascadia Mono', 'DejaVu Sans Mono', 'Symbols Nerd Font Mono', monospace",
}

/** When RAM and disk readings notify (internal/sysmon.Thresholds). */
export const ALERTS = {
  pct: { min: 50, max: 99, step: 5 },
  /** Seconds a reading must stay over before it notifies. */
  holds: [0, 60, 300, 900] as const,
  defaults: { mem: 90, disk: 85, hold: 60 },
} as const

/** Which files the Editor backs up before saving over them. */
export type BackupMode = 'etc' | 'always' | 'never'

/** A percent threshold kept in range; junk falls back to fallback. */
export function clampPct(v: number, fallback: number): number {
  if (!Number.isFinite(v)) return fallback
  return Math.min(ALERTS.pct.max, Math.max(ALERTS.pct.min, Math.round(v)))
}

interface SettingsData {
  reduceGlass: boolean
  reduceMotion: boolean
  wallpaperAccent: string
  terminalFontSize: number
  terminalGpu: boolean
  terminalPrompt: boolean
  monoFont: MonoFont
  alertMem: number
  alertDisk: number
  alertHold: number
  editorBackup: BackupMode
}

const defaults: SettingsData = {
  reduceGlass: false,
  reduceMotion: false,
  wallpaperAccent: WALLPAPER_ACCENTS[0],
  terminalFontSize: TERMINAL_FONT.default,
  terminalGpu: false,
  terminalPrompt: true,
  monoFont: 'jetbrains',
  alertMem: ALERTS.defaults.mem,
  alertDisk: ALERTS.defaults.disk,
  alertHold: ALERTS.defaults.hold,
  editorBackup: 'etc',
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
  get terminalPrompt() {
    return this.#data.terminalPrompt !== false
  }
  get terminalGpu() {
    return this.#data.terminalGpu === true
  }
  get terminalFontSize() {
    return clampFontSize(this.#data.terminalFontSize)
  }
  get monoFont(): MonoFont {
    return this.#data.monoFont in MONO_FAMILIES ? this.#data.monoFont : 'jetbrains'
  }
  /** The CSS font-family of monoFont. */
  get monoFamily() {
    return MONO_FAMILIES[this.monoFont]
  }
  get alertMem() {
    return clampPct(this.#data.alertMem, ALERTS.defaults.mem)
  }
  get alertDisk() {
    return clampPct(this.#data.alertDisk, ALERTS.defaults.disk)
  }
  get alertHold(): number {
    return (ALERTS.holds as readonly number[]).includes(this.#data.alertHold)
      ? this.#data.alertHold
      : ALERTS.defaults.hold
  }
  get editorBackup(): BackupMode {
    return ['etc', 'always', 'never'].includes(this.#data.editorBackup)
      ? this.#data.editorBackup
      : 'etc'
  }

  set<K extends keyof SettingsData>(key: K, value: SettingsData[K]) {
    this.#data[key] = value
    save('settings', this.#data)
  }
}

export const settings = new Settings()
