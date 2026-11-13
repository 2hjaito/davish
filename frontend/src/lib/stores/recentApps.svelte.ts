// The apps opened last, for Launchpad's "Recent" row. Kept with the other
// preferences (settings.json), across servers: it is about the user's habits.
import { load, save } from './persist'

/** One Launchpad row. */
export const RECENT_MAX = 5

/** list with id moved to the front, at most max long. */
export function pushRecent(list: readonly string[], id: string, max = RECENT_MAX): string[] {
  return [id, ...list.filter((x) => x !== id)].slice(0, max)
}

class RecentApps {
  // Read when the module loads, after settings.json (main.ts).
  ids = $state<string[]>(
    load<string[]>('recentApps', [])
      .filter((x) => typeof x === 'string')
      .slice(0, RECENT_MAX),
  )

  touch(id: string) {
    if (this.ids[0] === id) return
    this.ids = pushRecent(this.ids, id)
    save('recentApps', this.ids)
  }
}

export const recentApps = new RecentApps()
