// The person using Davish on this computer: a name and a picture, shown on
// Settings' account card and the lock screen. Kept in settings.json on this
// machine only; nothing here is sent anywhere.
import { load, save } from './persist'

export interface ProfileData {
  name: string
  /** A small square PNG data URL (shortcutLook.iconImage), or empty. */
  avatar: string
}

/** Longest name kept, so a paste cannot fill the sidebar. */
export const MAX_NAME = 64

/** Keeps a stored profile sane: strings only, a capped name, a PNG data URL. */
export function cleanProfile(raw: unknown): ProfileData {
  const r = raw && typeof raw === 'object' ? (raw as Record<string, unknown>) : {}
  const name = typeof r.name === 'string' ? r.name.trim().slice(0, MAX_NAME) : ''
  const avatar =
    typeof r.avatar === 'string' && r.avatar.startsWith('data:image/png;base64,') ? r.avatar : ''
  return { name, avatar }
}

class Profile {
  #data = $state<ProfileData>(cleanProfile(load<unknown>('profile', null)))

  get name() {
    return this.#data.name
  }

  get avatar() {
    return this.#data.avatar
  }

  set(patch: Partial<ProfileData>) {
    this.#data = cleanProfile({ ...this.#data, ...patch })
    save('profile', this.#data)
  }
}

export const profile = new Profile()
