// Turns a picture from this computer into the wallpaper that is stored: at
// most WALLPAPER_MAX_PX on the long side, as a JPEG. The picture never leaves
// this computer.
import { fitWithin, WALLPAPER_MAX_PX } from './wallpaper'

const MAX_SOURCE = 50 << 20
// SVG stays out: it can carry scripts, and it never meets the canvas below.
const TYPES = ['image/png', 'image/jpeg', 'image/webp']

export const WALLPAPER_ACCEPT = TYPES.join(',')

/** What the Go side will take (internal/wallpaper.MaxBytes), in characters of base64. */
const MAX_CHARS = Math.floor(((16 << 20) / 3) * 4) - 8

export async function wallpaperFromFile(file: File): Promise<string> {
  if (!TYPES.includes(file.type) || file.size > MAX_SOURCE) throw new Error('unsupported image')
  const bitmap = await createImageBitmap(file)
  try {
    const { w, h } = fitWithin(bitmap.width, bitmap.height, WALLPAPER_MAX_PX)
    if (w === 0) throw new Error('empty image')
    const canvas = document.createElement('canvas')
    canvas.width = w
    canvas.height = h
    const ctx = canvas.getContext('2d')
    if (!ctx) throw new Error('no canvas')
    // JPEG has no transparency: a see-through PNG lands on white, not black.
    ctx.fillStyle = '#fff'
    ctx.fillRect(0, 0, w, h)
    ctx.imageSmoothingQuality = 'high'
    ctx.drawImage(bitmap, 0, 0, w, h)
    for (const quality of [0.88, 0.75, 0.6]) {
      const url = canvas.toDataURL('image/jpeg', quality)
      if (url.length <= MAX_CHARS) return url
    }
    throw new Error('image too large')
  } finally {
    bitmap.close()
  }
}
