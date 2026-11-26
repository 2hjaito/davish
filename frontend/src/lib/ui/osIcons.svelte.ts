// The OS logos load the first time a server with a known system is drawn,
// not with the first screen (they are a chunk of their own).
import type { osIcons } from './osIcons'

class OsLogos {
  defs = $state.raw<typeof osIcons | null>(null)
  #started = false

  load() {
    if (this.#started) return
    this.#started = true
    void import('./osIcons').then((m) => {
      this.defs = m.osIcons
    })
  }
}

export const osLogos = new OsLogos()
