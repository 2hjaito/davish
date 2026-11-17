import {
  type Knobs,
  knobsFor,
  isNamedPreset,
  migratePerf,
  type NamedPreset,
  type PerfPreset,
  PRESET_KNOBS,
  presetMatching,
  type StoredPerf,
} from '../perf'
import { DEFAULT_STATS, STAT_IDS, type StatId } from '../shell/statIds'
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

/** The color scheme: follow the OS, or always light or dark. */
export type Theme = 'system' | 'light' | 'dark'

export const THEMES: readonly Theme[] = ['system', 'light', 'dark']

/** The scheme to draw: theme, with 'system' read from the OS. */
export function resolveTheme(theme: Theme, systemDark: boolean): 'light' | 'dark' {
  if (theme === 'system') return systemDark ? 'dark' : 'light'
  return theme
}

/** When RAM and disk readings notify (internal/sysmon.Thresholds). */
export const ALERTS = {
  pct: { min: 50, max: 99, step: 5 },
  /** Seconds a reading must stay over before it notifies. */
  holds: [0, 60, 300, 900] as const,
  defaults: { mem: 90, disk: 85, hold: 60 },
} as const

/** The screen edges the Dock can sit on; bottom is the default. */
export const DOCK_POSITIONS = ['bottom', 'left', 'right', 'top'] as const
export type DockPosition = (typeof DOCK_POSITIONS)[number]

/** The Dock's tile size, in px at rest. */
export const DOCK_SIZES = { small: 44, medium: 56, large: 68 } as const

export type DockSize = keyof typeof DOCK_SIZES

/** Which files the Editor backs up before saving over them. */
export type BackupMode = 'etc' | 'always' | 'never'

/** A percent threshold kept in range; junk falls back to fallback. */
export function clampPct(v: number, fallback: number): number {
  if (!Number.isFinite(v)) return fallback
  return Math.min(ALERTS.pct.max, Math.max(ALERTS.pct.min, Math.round(v)))
}

interface SettingsData extends StoredPerf {
  theme: Theme
  reduceGlass: boolean
  reduceMotion: boolean
  wallpaperAccent: string
  terminalFontSize: number
  terminalGpu: boolean
  terminalPrompt: boolean
  monoFont: MonoFont
  dockSize: DockSize
  dockAutohide: boolean
  dockPosition: DockPosition
  dockTransparent: boolean
  menubarStats: StatId[]
  alertMem: number
  alertDisk: number
  alertHold: number
  editorBackup: BackupMode
  lockAfter: number
  sudoForgetAfter: number
  autoUpdate: boolean
}

/** The choices for forgetting an unused sudo password, in minutes; 0 is never. */
export const SUDO_FORGET_AFTER = [5, 15, 60, 0]

const defaults: SettingsData = {
  theme: 'system',
  reduceGlass: false,
  reduceMotion: false,
  wallpaperAccent: WALLPAPER_ACCENTS[0],
  terminalFontSize: TERMINAL_FONT.default,
  terminalGpu: false,
  terminalPrompt: true,
  monoFont: 'jetbrains',
  dockSize: 'medium',
  dockAutohide: false,
  dockPosition: 'bottom',
  dockTransparent: false,
  menubarStats: [...DEFAULT_STATS],
  alertMem: ALERTS.defaults.mem,
  alertDisk: ALERTS.defaults.disk,
  alertHold: ALERTS.defaults.hold,
  editorBackup: 'etc',
  lockAfter: 15,
  sudoForgetAfter: 15,
  autoUpdate: true,
}

function fields(
  knobs: Knobs,
): Pick<
  SettingsData,
  | 'perfBlur'
  | 'perfShadow'
  | 'perfOpaque'
  | 'perfWallpaper'
  | 'perfMotion'
  | 'perfNoDmabuf'
  | 'perfNoCompositing'
  | 'reduceGlass'
  | 'reduceMotion'
> {
  return {
    perfBlur: knobs.blur,
    perfShadow: knobs.shadow,
    perfOpaque: knobs.opaque,
    perfWallpaper: knobs.wallpaper,
    perfMotion: knobs.motion,
    perfNoDmabuf: knobs.noDmabuf,
    perfNoCompositing: knobs.noCompositing,
    // The old toggles stay in the file so an older build still sees them.
    reduceGlass: !knobs.blur && knobs.opaque && knobs.shadow === 'short',
    reduceMotion: !knobs.motion,
  }
}

class Settings {
  #data = $state<SettingsData>(defaults)
  /** WebKit flags this process started with. A difference needs a restart. */
  #bootWebKit = { noDmabuf: false, noCompositing: false }

  constructor() {
    const raw = load<StoredPerf | null>('settings', null)
    const migrated = migratePerf(raw)
    const base = raw && typeof raw === 'object' ? { ...defaults, ...raw } : { ...defaults }
    this.#data = {
      ...base,
      perfPreset: migrated.preset,
      perfLater: migrated.later,
      ...fields(migrated.knobs),
    }
    this.#bootWebKit = {
      noDmabuf: migrated.knobs.noDmabuf,
      noCompositing: migrated.knobs.noCompositing,
    }
  }

  get theme(): Theme {
    return THEMES.includes(this.#data.theme) ? this.#data.theme : 'system'
  }
  /** True when the saved look is the old Reduce glass combination. */
  get reduceGlass() {
    return this.#data.reduceGlass === true
  }
  /** True when window and dock animations are off. */
  get reduceMotion() {
    return !this.knobs.motion
  }
  get perfPreset(): PerfPreset {
    return this.#data.perfPreset === 'custom' || isNamedPreset(this.#data.perfPreset)
      ? this.#data.perfPreset
      : 'auto'
  }
  /** The person dismissed the Light suggestion. */
  get perfLater() {
    return this.#data.perfLater === true
  }
  /** Knobs to paint. Auto paints Beautiful until a suggestion is applied. */
  get knobs(): Knobs {
    return knobsFor(this.perfPreset, {
      blur: this.#data.perfBlur !== false,
      shadow: this.#data.perfShadow === 'short' ? 'short' : 'deep',
      opaque: this.#data.perfOpaque === true,
      wallpaper: this.#data.perfWallpaper !== false,
      motion: this.#data.perfMotion !== false,
      noDmabuf: this.#data.perfNoDmabuf === true,
      noCompositing: this.#data.perfNoCompositing === true,
    })
  }
  /** Linux WebKit workarounds differ from the ones this process started with. */
  get perfNeedsRestart() {
    const k = this.knobs
    return (
      k.noDmabuf !== this.#bootWebKit.noDmabuf || k.noCompositing !== this.#bootWebKit.noCompositing
    )
  }

  /** Paints a named preset. Auto keeps Beautiful on screen. */
  usePreset(preset: NamedPreset) {
    this.#data.perfPreset = preset
    this.#data.perfLater = false
    Object.assign(this.#data, fields(knobsFor(preset, PRESET_KNOBS.beautiful)))
    save('settings', this.#data)
  }

  /** One advanced switch. Matching a preset snaps back to its name. */
  setKnob<K extends keyof Knobs>(key: K, value: Knobs[K]) {
    const next = { ...this.knobs, [key]: value }
    this.#data.perfPreset = presetMatching(next)
    Object.assign(this.#data, fields(next))
    save('settings', this.#data)
  }

  /** Hide the Light suggestion until Check again. */
  dismissPerfSuggestion() {
    this.#data.perfLater = true
    save('settings', this.#data)
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
  get dockSize(): DockSize {
    return this.#data.dockSize in DOCK_SIZES ? this.#data.dockSize : 'medium'
  }
  /** Slides the Dock off screen until the pointer reaches the bottom edge. */
  get dockAutohide() {
    return this.#data.dockAutohide === true
  }
  /** Which screen edge the Dock sits on. */
  get dockPosition(): DockPosition {
    return DOCK_POSITIONS.includes(this.#data.dockPosition) ? this.#data.dockPosition : 'bottom'
  }
  /** The Dock without its glass: only the icons float over the wallpaper. */
  get dockTransparent() {
    return this.#data.dockTransparent === true
  }
  /** The readings the menu bar shows; unknown ids from other versions drop out. */
  get menubarStats(): StatId[] {
    const ids = this.#data.menubarStats
    return Array.isArray(ids) ? STAT_IDS.filter((id) => ids.includes(id)) : [...DEFAULT_STATS]
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

  /** Look for a new Davish on GitHub on its own (ADR 0009). */
  get autoUpdate() {
    return this.#data.autoUpdate !== false
  }

  /** Idle minutes before the app locks; 0 is never. */
  get lockAfter(): number {
    return [0, 5, 15, 30, 60].includes(this.#data.lockAfter) ? this.#data.lockAfter : 15
  }

  /** Minutes an unused sudo password is kept, as sudo itself does; 0 is the whole session. */
  get sudoForgetAfter(): number {
    return SUDO_FORGET_AFTER.includes(this.#data.sudoForgetAfter) ? this.#data.sudoForgetAfter : 15
  }

  set<K extends keyof SettingsData>(key: K, value: SettingsData[K]) {
    this.#data[key] = value
    save('settings', this.#data)
  }
}

export const settings = new Settings()
