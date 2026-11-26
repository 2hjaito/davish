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

export const WALLPAPER_ACCENTS = [
  '#8fb2ff',
  '#c3a6ff',
  '#9fe0c8',
  '#ffb38a',
  '#ffa3c7',
  '#ffd97a',
  '#8fd8f0',
  '#c4c9d4',
] as const

/** A wallpaper color: one of the swatches or any color picked, as #rrggbb. */
const HEX_COLOR = /^#[0-9a-f]{6}$/i

export const TERMINAL_FONT = { min: 10, max: 20, step: 0.5, default: 12.5 } as const

/** Keeps a terminal font size in range and on the step grid; junk falls back to the default. */
export function clampFontSize(size: number): number {
  if (!Number.isFinite(size)) return TERMINAL_FONT.default
  const snapped = Math.round(size / TERMINAL_FONT.step) * TERMINAL_FONT.step
  return Math.min(TERMINAL_FONT.max, Math.max(TERMINAL_FONT.min, snapped))
}

/** How many lines a Terminal tab keeps to scroll back through, fewest first. */
export const TERMINAL_SCROLLBACK = [1000, 5000, 20000] as const
export type TerminalScrollback = (typeof TERMINAL_SCROLLBACK)[number]
const DEFAULT_SCROLLBACK: TerminalScrollback = 5000

/** One of the choices; anything else (junk from storage) is the default. */
export function clampScrollback(lines: unknown): TerminalScrollback {
  return TERMINAL_SCROLLBACK.find((n) => n === lines) ?? DEFAULT_SCROLLBACK
}

/** The color schemes of Terminal; their colors are in apps/terminal/schemes.ts. */
export const TERMINAL_SCHEMES = ['davish', 'phosphor', 'solarized', 'dracula'] as const
export type TerminalScheme = (typeof TERMINAL_SCHEMES)[number]

export function clampScheme(id: unknown): TerminalScheme {
  return TERMINAL_SCHEMES.find((s) => s === id) ?? 'davish'
}

/**
 * How a window follows the pointer while it is dragged. Live moves the whole
 * window. Simple does too, but paints the light look (no blur, a short shadow,
 * opaque fills) only until it is let go. Outline moves just an outline and puts
 * the window there on release, for machines that paint the window on the CPU.
 */
export const WINDOW_DRAGS = ['live', 'simple', 'outline'] as const
export type WindowDrag = (typeof WINDOW_DRAGS)[number]

export function clampDrag(mode: unknown): WindowDrag {
  return WINDOW_DRAGS.find((m) => m === mode) ?? 'live'
}

/** The shapes of the terminal cursor. */
export const TERMINAL_CURSORS = ['bar', 'block', 'underline'] as const
export type TerminalCursor = (typeof TERMINAL_CURSORS)[number]

export function clampCursor(shape: unknown): TerminalCursor {
  return TERMINAL_CURSORS.find((c) => c === shape) ?? 'bar'
}

/**
 * A font family name as typed, reduced to what a family name is made of
 * (letters, digits, space, dot, hyphen, underscore) so it can sit inside
 * quotes in a CSS font stack; anything else (a stray quote, a comma) is
 * dropped. Empty means "the monospace font of the whole app".
 */
export function cleanFontName(name: unknown): string {
  if (typeof name !== 'string') return ''
  return name
    .replace(/[^\p{L}\p{N} ._-]/gu, '')
    .replace(/\s+/g, ' ')
    .trim()
    .slice(0, 60)
}

export const EDITOR_FONT = { min: 10, max: 22, step: 0.5, default: 12.5 } as const

/** Keeps an editor font size in range and on the step grid; junk falls back to the default. */
export function clampEditorFontSize(size: unknown): number {
  if (typeof size !== 'number' || !Number.isFinite(size)) return EDITOR_FONT.default
  const snapped = Math.round(size / EDITOR_FONT.step) * EDITOR_FONT.step
  return Math.min(EDITOR_FONT.max, Math.max(EDITOR_FONT.min, snapped))
}

/** What the Tab key and auto-indent put in: two spaces, four spaces or a tab character. */
export const EDITOR_INDENTS = ['2', '4', 'tab'] as const
export type EditorIndent = (typeof EDITOR_INDENTS)[number]

export function clampIndent(indent: unknown): EditorIndent {
  return EDITOR_INDENTS.find((i) => i === indent) ?? '2'
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

/**
 * What Davish does when it opens (#273, #281): show All servers, open the
 * last server and connect to it, or open it without connecting.
 */
export const STARTUP_VIEWS = ['all', 'last', 'none'] as const
export type StartupView = (typeof STARTUP_VIEWS)[number]

/**
 * The first value of the setting. People who already have servers keep what
 * Davish always did; a new install opens on All servers.
 */
export function defaultStartup(hasServers: boolean): StartupView {
  return hasServers ? 'last' : 'all'
}

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
  terminalSolid: boolean
  terminalScheme: TerminalScheme
  terminalCursor: TerminalCursor
  windowDrag: WindowDrag
  outlineFill: boolean
  terminalBlink: boolean
  terminalFont: string
  terminalScrollback: TerminalScrollback
  terminalPrompt: boolean
  monoFont: MonoFont
  dockSize: DockSize
  dockAutohide: boolean
  dockPosition: DockPosition
  dockTransparent: boolean
  /** Null until the first launch of a build that has the setting. */
  startup: StartupView | null
  menubarStats: StatId[]
  alertMem: number
  alertDisk: number
  alertHold: number
  editorBackup: BackupMode
  editorFontSize: number
  editorWrap: boolean
  editorLineNumbers: boolean
  editorWhitespace: boolean
  editorIndent: EditorIndent
  filesShowHidden: boolean
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
  terminalSolid: false,
  terminalScheme: 'davish',
  terminalCursor: 'bar',
  windowDrag: 'live',
  outlineFill: true,
  terminalBlink: false,
  terminalFont: '',
  terminalScrollback: DEFAULT_SCROLLBACK,
  terminalPrompt: true,
  monoFont: 'jetbrains',
  dockSize: 'medium',
  dockAutohide: false,
  dockPosition: 'bottom',
  dockTransparent: false,
  startup: null,
  menubarStats: [...DEFAULT_STATS],
  alertMem: ALERTS.defaults.mem,
  alertDisk: ALERTS.defaults.disk,
  alertHold: ALERTS.defaults.hold,
  editorBackup: 'etc',
  editorFontSize: EDITOR_FONT.default,
  editorWrap: false,
  editorLineNumbers: true,
  editorWhitespace: false,
  editorIndent: '2',
  filesShowHidden: true,
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
  | 'perfWidgets'
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
    perfWidgets: knobs.widgets,
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
      widgets: this.#data.perfWidgets !== false,
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
    const c = this.#data.wallpaperAccent
    return typeof c === 'string' && HEX_COLOR.test(c) ? c.toLowerCase() : WALLPAPER_ACCENTS[0]
  }
  get terminalPrompt() {
    return this.#data.terminalPrompt !== false
  }
  get terminalGpu() {
    return this.#data.terminalGpu === true
  }
  get terminalSolid() {
    return this.#data.terminalSolid === true
  }
  /**
   * Whether the terminal draws on a solid background rather than the glass.
   * GPU drawing, the opaque performance setting and any scheme with a
   * background of its own always do.
   */
  get terminalOpaque() {
    return (
      this.terminalGpu ||
      this.terminalSolid ||
      this.knobs.opaque ||
      this.terminalScheme !== 'davish'
    )
  }
  get terminalScheme(): TerminalScheme {
    return clampScheme(this.#data.terminalScheme)
  }
  get terminalCursor(): TerminalCursor {
    return clampCursor(this.#data.terminalCursor)
  }
  get windowDrag(): WindowDrag {
    return clampDrag(this.#data.windowDrag)
  }
  /** Outline mode: a light tint inside the outline. Off draws the border alone, the lightest. */
  get outlineFill() {
    return this.#data.outlineFill !== false
  }
  get terminalBlink() {
    return this.#data.terminalBlink === true
  }
  /** The family the person named for Terminal only; empty for the app's monospace font. */
  get terminalFont() {
    return cleanFontName(this.#data.terminalFont)
  }
  /** The CSS font-family of Terminal: the named font if installed, then monoFamily. */
  get terminalFontFamily() {
    const name = this.terminalFont
    return name ? `'${name}', ${this.monoFamily}` : this.monoFamily
  }
  get terminalScrollback(): TerminalScrollback {
    return clampScrollback(this.#data.terminalScrollback)
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
  /** What opens at launch; null before the first launch has chosen one. */
  get startup(): StartupView | null {
    const v = this.#data.startup
    return v !== null && STARTUP_VIEWS.includes(v) ? v : null
  }
  /**
   * Picks the launch behavior once, on the first launch that has the setting:
   * people who already have servers keep opening the last one.
   */
  settleStartup(hasServers: boolean): StartupView {
    const current = this.startup
    if (current) return current
    const v = defaultStartup(hasServers)
    this.set('startup', v)
    return v
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
  get editorFontSize() {
    return clampEditorFontSize(this.#data.editorFontSize)
  }
  /** Long lines continue on the next row instead of scrolling sideways. */
  get editorWrap() {
    return this.#data.editorWrap === true
  }
  get editorLineNumbers() {
    return this.#data.editorLineNumbers !== false
  }
  /** Spaces and tabs drawn as marks. */
  get editorWhitespace() {
    return this.#data.editorWhitespace === true
  }
  get editorIndent(): EditorIndent {
    return clampIndent(this.#data.editorIndent)
  }
  /** Files lists names that start with a dot. */
  get filesShowHidden() {
    return this.#data.filesShowHidden !== false
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
