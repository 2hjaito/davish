/** How Davish spends GPU and CPU on its own window. Named presets set every knob. */

export const NAMED_PRESETS = ['auto', 'beautiful', 'balanced', 'light'] as const

export type NamedPreset = (typeof NAMED_PRESETS)[number]

export type PerfPreset = NamedPreset | 'custom'

export type Shadow = 'deep' | 'short'

/** The switches a preset is made of. WebKit flags stay off in every named preset. */
export interface Knobs {
  /** Frosted blur behind windows. */
  blur: boolean
  shadow: Shadow
  /** Fully opaque window and content fills. */
  opaque: boolean
  /** Colored wallpaper blobs. False is a flat base color. */
  wallpaper: boolean
  /** Open, close and minimize animations. */
  motion: boolean
  /** Linux: set WEBKIT_DISABLE_DMABUF_RENDERER on the next launch. */
  noDmabuf: boolean
  /** Linux: set WEBKIT_DISABLE_COMPOSITING_MODE on the next launch. */
  noCompositing: boolean
}

export const PRESET_KNOBS: Record<Exclude<NamedPreset, 'auto'>, Knobs> = {
  beautiful: {
    blur: true,
    shadow: 'deep',
    opaque: false,
    wallpaper: true,
    motion: true,
    noDmabuf: false,
    noCompositing: false,
  },
  balanced: {
    blur: false,
    shadow: 'deep',
    opaque: true,
    wallpaper: true,
    motion: true,
    noDmabuf: false,
    noCompositing: false,
  },
  light: {
    blur: false,
    shadow: 'short',
    opaque: true,
    wallpaper: false,
    motion: false,
    noDmabuf: false,
    noCompositing: false,
  },
}

const SLOW_RENDERER = /llvmpipe|softpipe|swiftshader/i

/** Frames slower than this (p95) count as a slow machine when the renderer name is a GPU. */
export const SLOW_FRAME_MS = 28

/** How long the first-launch check watches frames. It does not block the UI. */
export const PROBE_MS = 1500

export function isNamedPreset(v: unknown): v is NamedPreset {
  return typeof v === 'string' && (NAMED_PRESETS as readonly string[]).includes(v)
}

export function knobsEqual(a: Knobs, b: Knobs): boolean {
  return (
    a.blur === b.blur &&
    a.shadow === b.shadow &&
    a.opaque === b.opaque &&
    a.wallpaper === b.wallpaper &&
    a.motion === b.motion &&
    a.noDmabuf === b.noDmabuf &&
    a.noCompositing === b.noCompositing
  )
}

/** The named preset these knobs are, or custom when none match. */
export function presetMatching(knobs: Knobs): PerfPreset {
  for (const name of ['beautiful', 'balanced', 'light'] as const) {
    if (knobsEqual(PRESET_KNOBS[name], knobs)) return name
  }
  return 'custom'
}

/** What to paint. Auto uses Beautiful until the person applies a suggestion. */
export function knobsFor(preset: PerfPreset, custom: Knobs): Knobs {
  if (preset === 'custom') return custom
  if (preset === 'auto') return PRESET_KNOBS.beautiful
  return PRESET_KNOBS[preset]
}

export interface StoredPerf {
  perfPreset?: unknown
  perfLater?: unknown
  perfBlur?: unknown
  perfShadow?: unknown
  perfOpaque?: unknown
  perfWallpaper?: unknown
  perfMotion?: unknown
  perfNoDmabuf?: unknown
  perfNoCompositing?: unknown
  /** Previous Reduce glass toggle, before presets. */
  reduceGlass?: unknown
  reduceMotion?: unknown
}

function flag(v: unknown, fallback: boolean): boolean {
  return typeof v === 'boolean' ? v : fallback
}

export function knobsFromStored(raw: StoredPerf): Knobs {
  return {
    blur: flag(raw.perfBlur, true),
    shadow: raw.perfShadow === 'short' ? 'short' : 'deep',
    opaque: flag(raw.perfOpaque, false),
    wallpaper: flag(raw.perfWallpaper, true),
    motion: flag(raw.perfMotion, true),
    noDmabuf: flag(raw.perfNoDmabuf, false),
    noCompositing: flag(raw.perfNoCompositing, false),
  }
}

export interface ResolvedPerf {
  preset: PerfPreset
  knobs: Knobs
  /** The person chose Later on the Light suggestion. */
  later: boolean
}

/**
 * A saved preset wins. A file from before presets keeps Reduce glass and
 * Reduce motion as knobs (that combination is custom, not Light). A new
 * install is Auto, which paints Beautiful until a suggestion is applied.
 */
export function migratePerf(raw: StoredPerf | null): ResolvedPerf {
  const later = raw?.perfLater === true
  if (raw && isNamedPreset(raw.perfPreset)) {
    if (raw.perfPreset === 'auto') {
      return { preset: 'auto', knobs: { ...PRESET_KNOBS.beautiful }, later }
    }
    return { preset: raw.perfPreset, knobs: { ...PRESET_KNOBS[raw.perfPreset] }, later }
  }
  if (raw?.perfPreset === 'custom') {
    return { preset: 'custom', knobs: knobsFromStored(raw), later }
  }
  if (raw && (raw.reduceGlass === true || raw.reduceMotion === true)) {
    const knobs: Knobs = {
      blur: raw.reduceGlass !== true,
      shadow: raw.reduceGlass === true ? 'short' : 'deep',
      opaque: raw.reduceGlass === true,
      wallpaper: true,
      motion: raw.reduceMotion !== true,
      noDmabuf: false,
      noCompositing: false,
    }
    return { preset: presetMatching(knobs), knobs, later: false }
  }
  return { preset: 'auto', knobs: { ...PRESET_KNOBS.beautiful }, later: false }
}

/** p95 of frame gaps. An empty sample is 0. */
export function frameP95(samples: readonly number[]): number {
  if (samples.length === 0) return 0
  const sorted = [...samples].sort((a, b) => a - b)
  return sorted[Math.min(sorted.length - 1, Math.floor(0.95 * (sorted.length - 1)))] ?? 0
}

/**
 * Slow when the WebGL renderer is a software rasterizer, or when enough
 * frames were timed and the tail is slower than SLOW_FRAME_MS. A short or
 * empty sample does not by itself count as slow.
 */
export function machineIsSlow(renderer: string, samples: readonly number[]): boolean {
  if (SLOW_RENDERER.test(renderer)) return true
  if (samples.length < 20) return false
  return frameP95(samples) > SLOW_FRAME_MS
}

/** Frames up to this (p95) keep the Beautiful look smooth: about 50 fps or better. */
export const SMOOTH_FRAME_MS = 20

/**
 * The look this machine draws smoothly, timed with Beautiful on: slow is
 * Light, a tail between smooth and slow is Balanced, otherwise Beautiful.
 */
export function recommendPreset(
  renderer: string,
  samples: readonly number[],
): Exclude<NamedPreset, 'auto'> {
  if (machineIsSlow(renderer, samples)) return 'light'
  if (samples.length >= 20 && frameP95(samples) > SMOOTH_FRAME_MS) return 'balanced'
  return 'beautiful'
}

/** Frames per second from the median frame gap, rounded. 0 for no sample. */
export function framesPerSecond(samples: readonly number[]): number {
  if (samples.length === 0) return 0
  const sorted = [...samples].sort((a, b) => a - b)
  const median = sorted[Math.floor(sorted.length / 2)] ?? 0
  return median > 0 ? Math.round(1000 / median) : 0
}
