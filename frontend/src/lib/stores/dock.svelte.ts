import { dockApps } from '../../apps/registry'
import { togglePin } from '../shell/dockItems'
import { load, save } from './persist'

/** The apps pinned to the dock, in order; Launchpad pins and unpins. */
class DockPins {
  ids = $state<string[]>(
    load(
      'dock:pinned',
      dockApps.map((a) => a.id),
    ),
  )

  has(id: string): boolean {
    return this.ids.includes(id)
  }

  toggle(id: string) {
    this.ids = togglePin(this.ids, id)
    save('dock:pinned', this.ids)
  }
}

export const dockPins = new DockPins()
