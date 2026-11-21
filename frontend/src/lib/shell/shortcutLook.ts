// How a desktop icon looks: what the user may pick, and what falls back to
// the file behind it. Nothing here touches the server.
import type { Shortcut } from '../stores/shortcuts.svelte'
import type { IconName } from '../ui/icons'

const KIND_ICONS: Record<Shortcut['kind'], IconName> = {
  script: 'terminal',
  file: 'file',
  folder: 'folder',
}

export const ICON_CHOICES = [
  'terminal',
  'folder',
  'file',
  'play',
  'zap',
  'package',
  'server',
  'disk',
  'globe2',
  'gitBranch',
  'shield',
  'key',
  'clock',
  'home',
  'settings',
  'monitor',
] as const satisfies readonly IconName[]

/** The app colors of .claude/docs/DESIGN.md, so a tinted icon sits well in the dock's company. */
export const COLOR_CHOICES = {
  graphite: '#24262d',
  blue: '#2f7de1',
  green: '#16a067',
  orange: '#e08a12',
  teal: '#14a3a8',
  purple: '#7a52d6',
  red: '#d6455a',
  gray: '#6b7180',
} as const

export type IconChoice = (typeof ICON_CHOICES)[number]
export type ColorChoice = keyof typeof COLOR_CHOICES

/** The name to show: the user's, or the file's. */
export function titleOf(s: Pick<Shortcut, 'label' | 'name'>): string {
  return s.label ?? s.name
}

export function iconOf(s: Pick<Shortcut, 'icon' | 'kind'>): IconName {
  return s.icon ?? KIND_ICONS[s.kind]
}

/** A label that repeats the file's name, or is blank, is no label. */
export function labelFor(typed: string, name: string): string | undefined {
  const label = typed.trim()
  return label && label !== name ? label : undefined
}

// Twice the 52px tile, for HiDPI screens. At this size a PNG of pure noise
// still fits the 64 KiB a settings key may hold (internal/store).
const ICON_PX = 104
const MAX_SOURCE = 20 << 20
// SVG stays out: it can carry scripts, and it never meets the canvas below.
const IMAGE_TYPES = ['image/png', 'image/jpeg', 'image/webp']

export const IMAGE_ACCEPT = IMAGE_TYPES.join(',')

/** The largest centered square of a picture. */
export function squareCrop(width: number, height: number) {
  const side = Math.min(width, height)
  return { x: (width - side) / 2, y: (height - side) / 2, side }
}

/** Turns a picture from this computer into a small PNG data URL. */
export async function iconImage(file: File): Promise<string> {
  if (!IMAGE_TYPES.includes(file.type) || file.size > MAX_SOURCE) {
    throw new Error('unsupported image')
  }
  const bitmap = await createImageBitmap(file)
  const { x, y, side } = squareCrop(bitmap.width, bitmap.height)
  const canvas = document.createElement('canvas')
  canvas.width = canvas.height = ICON_PX
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('no canvas')
  ctx.imageSmoothingQuality = 'high'
  ctx.drawImage(bitmap, x, y, side, side, 0, 0, ICON_PX, ICON_PX)
  bitmap.close()
  return canvas.toDataURL('image/png')
}

// The account picture: sharp at 88px on a 2x screen with room to spare, as a
// JPEG so a photo stays around 30 KB. The profile key, name included, must
// stay under the 64 KiB a settings key may hold (internal/store).
const AVATAR_PX = 256
const AVATAR_MAX_CHARS = 56_000

/** Turns a picture from this computer into a square JPEG data URL for the account. */
export async function avatarImage(file: File): Promise<string> {
  if (!IMAGE_TYPES.includes(file.type) || file.size > MAX_SOURCE) {
    throw new Error('unsupported image')
  }
  const bitmap = await createImageBitmap(file)
  try {
    const { x, y, side } = squareCrop(bitmap.width, bitmap.height)
    for (const [px, quality] of [
      [AVATAR_PX, 0.88],
      [AVATAR_PX, 0.72],
      [192, 0.6],
    ] as const) {
      const canvas = document.createElement('canvas')
      canvas.width = canvas.height = px
      const ctx = canvas.getContext('2d')
      if (!ctx) throw new Error('no canvas')
      // JPEG has no transparency: a see-through PNG lands on white, not black.
      ctx.fillStyle = '#fff'
      ctx.fillRect(0, 0, px, px)
      ctx.imageSmoothingQuality = 'high'
      ctx.drawImage(bitmap, x, y, side, side, 0, 0, px, px)
      const url = canvas.toDataURL('image/jpeg', quality)
      if (url.length <= AVATAR_MAX_CHARS) return url
    }
    throw new Error('image too large')
  } finally {
    bitmap.close()
  }
}
