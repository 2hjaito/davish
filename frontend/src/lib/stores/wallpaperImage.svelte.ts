// The picture chosen as wallpaper. It is far too big for settings.json, so
// Go keeps it in a file of its own (internal/wallpaper) and this store holds
// the data URL once it is needed. Nothing is read until something asks.
import { wallpaperClear, wallpaperGet, wallpaperSet } from '../api/wallpaper'

class WallpaperImage {
  #url = $state('')
  #loaded = $state(false)
  #loading: Promise<void> | null = null

  /** The picture as a data URL; "" before it loads or when there is none. */
  get url() {
    return this.#url
  }

  /** True once the file was read, so "none" can be told from "not yet". */
  get loaded() {
    return this.#loaded
  }

  get has() {
    return this.#url !== ''
  }

  /** Reads the picture once; later calls share that read. */
  ensure(): Promise<void> {
    this.#loading ??= wallpaperGet()
      .then((u) => {
        this.#url = u
      })
      .catch(() => {
        // A picture that cannot be read leaves the colored wallpaper.
      })
      .finally(() => {
        this.#loaded = true
      })
    return this.#loading
  }

  async set(dataURL: string): Promise<void> {
    await wallpaperSet(dataURL)
    this.#url = dataURL
    this.#loaded = true
  }

  async clear(): Promise<void> {
    await wallpaperClear()
    this.#url = ''
    this.#loaded = true
  }
}

export const wallpaperImage = new WallpaperImage()
