import { DEFAULT_PINS, togglePin } from '../shell/dockItems'
import { load, save } from './persist'

/** The apps pinned to the dock, in order; Launchpad pins and unpins. */
class DockPins {
  // Saved only once the user pins or unpins, so their own dock stays.
  ids = $state<string[]>(load('dock:pinned', [...DEFAULT_PINS]))

  has(id: string): boolean {
    return this.ids.includes(id)
  }

  toggle(id: string) {
    this.ids = togglePin(this.ids, id)
    save('dock:pinned', this.ids)
  }

  /** True while the dock holds the first-run pins, in their order. */
  get isDefault(): boolean {
    return this.ids.join() === DEFAULT_PINS.join()
  }

  reset() {
    this.ids = [...DEFAULT_PINS]
    save('dock:pinned', this.ids)
  }
}

export const dockPins = new DockPins()
