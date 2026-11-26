// Whether the picture behind the menu bar is dark or light, so the bar can
// pick light or dark text for it, as macOS does (#297 follow-up). The theme
// decides everything else; over a picture only the picture can tell.

export type Tone = 'dark' | 'light'

/**
 * Below this relative luminance the bar uses light text. It is where black
 * and white text have the same contrast (about 0.18), a little higher because
 * the bar's own glass tints the picture.
 */
export const TONE_THRESHOLD = 0.2

/** How much of the picture, from the top, the menu bar covers. */
export const BAR_STRIP = 0.08

const channel = (v: number) => {
  const s = v / 255
  return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4
}

/** WCAG relative luminance of an sRGB color: 0 is black, 1 is white. */
export function luminance(r: number, g: number, b: number): number {
  return 0.2126 * channel(r) + 0.7152 * channel(g) + 0.0722 * channel(b)
}

/** The mean luminance of RGBA pixel data; 0.5 for no pixels. */
export function meanLuminance(data: ArrayLike<number>): number {
  const n = Math.floor(data.length / 4)
  if (n === 0) return 0.5
  let sum = 0
  for (let i = 0; i < n; i++)
    sum += luminance(data[i * 4] ?? 0, data[i * 4 + 1] ?? 0, data[i * 4 + 2] ?? 0)
  return sum / n
}

/**
 * The tone of a picture with this mean luminance under a dim layer of dim
 * percent (a black layer of that opacity): dimming a picture makes it darker.
 */
export function toneOf(luma: number, dim = 0): Tone {
  const shown = luma * (1 - Math.min(100, Math.max(0, dim)) / 100)
  return shown < TONE_THRESHOLD ? 'dark' : 'light'
}

/**
 * The mean luminance of the top strip of a picture, read from a 64 px copy
 * of that strip: nothing is kept and it costs nothing after the one read.
 * Null where the picture cannot be read.
 */
export async function sampleBarLuminance(dataURL: string): Promise<number | null> {
  try {
    const img = new Image()
    img.src = dataURL
    await img.decode()
    if (!img.naturalWidth || !img.naturalHeight) return null
    const w = 64
    const h = 6
    const c = document.createElement('canvas')
    c.width = w
    c.height = h
    const g = c.getContext('2d', { willReadFrequently: true })
    if (!g) return null
    g.drawImage(img, 0, 0, img.naturalWidth, img.naturalHeight * BAR_STRIP, 0, 0, w, h)
    return meanLuminance(g.getImageData(0, 0, w, h).data)
  } catch {
    return null
  }
}
