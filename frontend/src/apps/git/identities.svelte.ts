// Who each server works as for git, kept for the session so reopening Git
// asks nothing; refresh asks again. The quick part (user, git config, keys)
// shows at once. Each code host is asked on its own, and fills in when it
// answers: a server that never reaches GitLab does not hold up GitHub.
import { toUserError } from '../../lib/api/errors'
import { CODE_HOSTS, type HostLogin, type Identity } from '../../lib/api/git'
import { backend } from './backend'

export interface IdentityEntry {
  identity: Identity | null
  /** Hosts still being asked. */
  pending: string[]
  error: string
}

const EMPTY: IdentityEntry = { identity: null, pending: [], error: '' }

class Identities {
  #byServer = $state<Record<string, IdentityEntry>>({})

  get(serverID: string): IdentityEntry {
    return this.#byServer[serverID] ?? EMPTY
  }

  #patch(serverID: string, patch: Partial<IdentityEntry>) {
    this.#byServer[serverID] = { ...this.get(serverID), ...patch }
  }

  /** Reads serverID's identity unless already read; force reads it again. */
  async load(serverID: string, force = false) {
    const now = this.#byServer[serverID]
    if (now?.pending.length) return
    // A read that failed is tried again, as when the server comes back.
    if (now?.identity && !force) return
    this.#patch(serverID, { pending: [...CODE_HOSTS], error: '' })
    try {
      const id = await backend.identity(serverID)
      // On a refresh the last logins stay until their host answers again.
      this.#patch(serverID, {
        identity: { ...id, hosts: this.get(serverID).identity?.hosts ?? [] },
      })
    } catch (e) {
      this.#patch(serverID, { error: toUserError(e).message, pending: [] })
      return
    }
    await Promise.all(CODE_HOSTS.map((h) => this.#ask(serverID, h)))
  }

  async #ask(serverID: string, host: string) {
    let login: HostLogin = { host, state: 'unreachable', account: '', deploy: false }
    try {
      login = await backend.hostLogin(serverID, host)
    } catch {
      // A host that cannot be asked is simply not shown.
    }
    const cur = this.get(serverID)
    const hosts = [...(cur.identity?.hosts ?? []).filter((h) => h.host !== host), login]
    this.#patch(serverID, {
      identity: cur.identity ? { ...cur.identity, hosts } : null,
      pending: cur.pending.filter((h) => h !== host),
    })
  }
}

export const identities = new Identities()
