import { onConnStatus, onHostKeyPrompt, type HostKeyPrompt } from '../api/events'
import { type ErrorCode, toUserError } from '../api/errors'
import { hasRuntime } from '../api/runtime'
import * as api from '../api/servers'
import type { ConnState, ServerDraft, ServerInfo } from '../api/servers'
import { load, save } from './persist'

export type { ConnState }

export interface Server extends ServerInfo {
  /** Why the last attempt failed, when state is 'failed'. */
  error?: ErrorCode
}

// Shown in a plain browser (vite dev, previews), where there is no Go side.
const demoServers: Server[] = [
  {
    id: 'tsc-vps',
    name: 'tsc-vps',
    host: 'vps.example.com',
    port: 22,
    user: 'dangth',
    auth: 'agent',
    hasPassword: false,
    hasPassphrase: false,
    state: 'connected',
  },
  {
    id: 'nas',
    name: 'NAS văn phòng',
    host: '192.168.1.20',
    port: 22,
    user: 'dangth',
    auth: 'password',
    hasPassword: true,
    hasPassphrase: false,
    state: 'disconnected',
  },
]

class Servers {
  list = $state<Server[]>(hasRuntime ? [] : demoServers)
  activeId = $state(load('activeServer', { id: '' }).id || (hasRuntime ? '' : 'tsc-vps'))
  /** A host key waiting for the user to accept it. */
  hostKeyPrompt = $state<HostKeyPrompt | null>(null)
  /** True once the list has been loaded from Go (always true in a browser). */
  ready = $state(!hasRuntime)

  active = $derived(this.list.find((s) => s.id === this.activeId))

  /** Loads the saved servers and starts listening to connection events. */
  async init(): Promise<() => void> {
    if (!hasRuntime) return () => {}
    const offStatus = onConnStatus((st) => {
      const s = this.list.find((x) => x.id === st.serverID)
      if (!s) return
      s.state = st.state
      s.error = st.error ? toUserError(st.error).code : undefined
    })
    const offPrompt = onHostKeyPrompt((p) => (this.hostKeyPrompt = p))
    await this.reload()
    this.ready = true
    if (!this.active) this.activeId = this.list[0]?.id ?? ''
    void this.ensureConnected(this.activeId)
    return () => {
      offStatus()
      offPrompt()
    }
  }

  async reload() {
    this.list = await api.listServers()
  }

  select(id: string) {
    if (!this.list.some((s) => s.id === id)) return
    this.activeId = id
    save('activeServer', { id })
    void this.ensureConnected(id)
  }

  selectIndex(i: number) {
    const s = this.list[i]
    if (s) this.select(s.id)
  }

  /** Connects a server that is idle or failed; errors land on the server. */
  async ensureConnected(id: string) {
    const s = this.list.find((x) => x.id === id)
    if (!hasRuntime || !s || (s.state !== 'disconnected' && s.state !== 'failed')) return
    try {
      await api.connect(id)
    } catch (e) {
      // conn:status already marked it failed; keep the reason for the UI.
      s.error = toUserError(e).code
    }
  }

  /** Saves a new or edited server and returns it; throws a UserError. */
  async save(draft: ServerDraft): Promise<Server> {
    const saved = await api.saveServer(draft)
    await this.reload()
    return saved
  }

  async remove(id: string) {
    await api.deleteServer(id)
    await this.reload()
    if (this.activeId === id) this.select(this.list[0]?.id ?? '')
  }

  async reorder(ids: string[]) {
    // Optimistic, so dragging feels instant; reload restores on failure.
    const byId = new Map(this.list.map((s) => [s.id, s]))
    this.list = ids.flatMap((id) => byId.get(id) ?? [])
    try {
      await api.reorderServers(ids)
    } catch (e) {
      await this.reload()
      throw e
    }
  }

  async reconnect(id: string) {
    const s = this.list.find((x) => x.id === id)
    if (s) s.state = 'disconnected'
    await this.ensureConnected(id)
  }

  async disconnect(id: string) {
    if (hasRuntime) await api.disconnect(id)
  }

  /** Trusts the prompted host key and connects. */
  async acceptHostKey() {
    const p = this.hostKeyPrompt
    this.hostKeyPrompt = null
    if (p) await api.acceptHostKey(p.serverID, p.fingerprint)
  }

  rejectHostKey() {
    this.hostKeyPrompt = null
  }
}

export const servers = new Servers()
