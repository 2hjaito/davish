/** #rrggbb to channels. Anything else is rejected. */
function channels(hex: string): [number, number, number] | null {
  const m = /^#([0-9a-f]{6})$/i.exec(hex.trim())
  if (!m?.[1]) return null
  const n = Number.parseInt(m[1], 16)
  return [(n >> 16) & 255, (n >> 8) & 255, n & 255]
}

/**
 * The flat wallpaper: the chosen swatch mixed into the base by the same
 * amount the blobs use (--wallpaper-mix). Light is 100%, so the swatch is
 * the color; dark keeps it dimmed.
 */
export function mixWallpaper(accent: string, base: string, mixPercent: number): string | null {
  const a = channels(accent)
  const b = channels(base)
  if (!a || !b || !Number.isFinite(mixPercent)) return null
  const t = Math.min(100, Math.max(0, mixPercent)) / 100
  const hex = (n: number) => Math.round(n).toString(16).padStart(2, '0')
  return `#${hex(a[0] * t + b[0] * (1 - t))}${hex(a[1] * t + b[1] * (1 - t))}${hex(a[2] * t + b[2] * (1 - t))}`
}

/** What the desktop paints behind everything: the picture, the colored blobs, or one flat color. */
export type WallpaperMode = 'image' | 'blobs' | 'flat'

/**
 * The picture wins when the person chose one and the preset allows it. Without
 * it the wallpaper knob decides, as it did before pictures. A preset that turns
 * the picture off (Light) falls back to a flat color, not to blobs it also turns off.
 */
export function wallpaperMode(
  knobs: { wallpaper: boolean; wallpaperImage: boolean },
  hasImage: boolean,
): WallpaperMode {
  if (hasImage && knobs.wallpaperImage) return 'image'
  return knobs.wallpaper ? 'blobs' : 'flat'
}

/** Longest side of a stored picture: sharp on a 2560px display, small enough to keep and to decode. */
export const WALLPAPER_MAX_PX = 2560

/** The size to draw a picture at so neither side passes max. Never enlarges. */
export function fitWithin(width: number, height: number, max: number): { w: number; h: number } {
  if (!(width > 0) || !(height > 0)) return { w: 0, h: 0 }
  const scale = Math.min(1, max / Math.max(width, height))
  return { w: Math.max(1, Math.round(width * scale)), h: Math.max(1, Math.round(height * scale)) }
}
