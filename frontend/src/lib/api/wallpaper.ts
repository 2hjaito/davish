// Typed wrappers for the wallpaper bindings (internal/app/wallpaper.go).
// In a plain browser (vite dev, previews, demo mode) there is no Go side, so
// the picture is kept in memory and gone on reload.
import * as App from '../../../wailsjs/go/app/App'
import { hasRuntime } from './runtime'

let preview = ''

/** The chosen picture as a data URL, or "" when there is none. */
export const wallpaperGet = (): Promise<string> =>
  hasRuntime ? App.WallpaperGet() : Promise.resolve(preview)

export async function wallpaperSet(dataURL: string): Promise<void> {
  if (hasRuntime) return App.WallpaperSet(dataURL)
  preview = dataURL
}

export async function wallpaperClear(): Promise<void> {
  if (hasRuntime) return App.WallpaperClear()
  preview = ''
}
