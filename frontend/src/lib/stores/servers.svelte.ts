import { onConnStatus, onHostKeyPrompt, type HostKeyPrompt } from '../api/events'
import { type ErrorCode, toUserError } from '../api/errors'
import { hasRuntime } from '../api/runtime'
import * as api from '../api/servers'
import { t } from '../i18n'
import type { ConnState, Platform, ServerDraft, ServerInfo } from '../api/servers'
import { load, save } from './persist'
import { shortcuts } from './shortcuts.svelte'

export type { ConnState, Platform }

export interface Server extends ServerInfo {
  /** Why the last attempt failed, when state is 'failed'. */
  error?: ErrorCode
  /** The backend's English message and technical cause of error, for a bug report. */
  errorMessage?: string
  errorDetail?: string
  /** The server's OS, known once it has connected; kept across reconnects. */
  platform?: Platform
}

// Shown in a plain browser (vite dev, previews), where there is no Go side.
const demoServers: Server[] = [
  {
    id: 'dangth-vps',
    name: 'dangth-vps',
    host: 'vps.dangth.dev',
    port: 22,
    user: 'dangth',
    auth: 'agent',
    hasPassword: false,
    hasPassphrase: false,
    state: 'connected',
  },
  {
    id: 'nas',
    name: t.demo.nas,
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
  // The browser preview shows the first-run welcome page with ?welcome.
  list = $state<Server[]>(
    hasRuntime || (typeof location !== 'undefined' && location.search.includes('welcome'))
      ? []
      : demoServers,
  )
  activeId = $state(load('activeServer', { id: '' }).id || (hasRuntime ? '' : 'dangth-vps'))
  /** A host key waiting for the user to accept it. */
  hostKeyPrompt = $state<HostKeyPrompt | null>(null)
  /** The server whose failed connection the error dialog shows. */
  failureOf = $state<string | null>(null)
  /** True once the list has been loaded from Go (always true in a browser). */
  ready = $state(!hasRuntime)

  active = $derived(this.list.find((s) => s.id === this.activeId))

  /** The server's OS; 'unknown' until it has connected. */
  platformOf(id: string): Platform {
    return this.list.find((s) => s.id === id)?.platform ?? 'unknown'
  }

  /** Loads the saved servers and starts listening to connection events. */
  async init(): Promise<() => void> {
    if (!hasRuntime) return () => {}
    const offStatus = onConnStatus((st) => {
      const s = this.list.find((x) => x.id === st.serverID)
      if (!s) return
      s.state = st.state
      if (st.platform) s.platform = st.platform
      s.error = st.error ? toUserError(st.error).code : undefined
      s.errorMessage = st.error?.message
      s.errorDetail = st.detail
      // A new host key has its own dialog; every other failure opens the
      // error dialog, so the cause can be read and copied.
      if (st.state === 'failed' && s.error !== 'CONN_HOSTKEY_UNKNOWN') this.failureOf = s.id
      else if (this.failureOf === s.id && st.state !== 'failed') this.failureOf = null
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
    s.error = s.errorMessage = s.errorDetail = undefined
    try {
      await api.connect(id)
    } catch (e) {
      // conn:status already marked it failed with the cause; keep the
      // reason when the event did not carry one.
      if (!s.error) {
        const ue = toUserError(e)
        s.error = ue.code
        s.errorMessage = ue.detail
      }
    }
  }

  /** Saves a new or edited server and returns it; throws a UserError. */
  async save(draft: ServerDraft): Promise<Server> {
    const saved = await api.saveServer(draft)
    await this.reload()
    return saved
  }

  /** Imports ~/.ssh/config hosts and returns the servers added; throws a UserError. */
  async importHosts(aliases: string[]): Promise<Server[]> {
    try {
      return await api.importConfigHosts(aliases)
    } finally {
      await this.reload()
    }
  }

  async remove(id: string) {
    await api.deleteServer(id)
    shortcuts.forgetServer(id)
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

  /** Opens the error dialog for a failed server. */
  showFailure(id: string) {
    this.failureOf = id
  }

  closeFailure() {
    this.failureOf = null
  }
}

export const servers = new Servers()
