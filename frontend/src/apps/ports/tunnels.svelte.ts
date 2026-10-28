// Running tunnels, shared by every Ports window and kept current by
// tunnel:changed. They end with the app; nothing is saved.
import { onTunnelChanged } from '../../lib/api/events'
import type { Tunnel } from '../../lib/api/ports'
import { hasRuntime } from '../../lib/api/runtime'
import { backend } from './backend'

class Tunnels {
  list = $state.raw<Tunnel[]>([])
  #started = false

  /** Loads the list once and follows changes for the app's lifetime. */
  init() {
    if (this.#started) return
    this.#started = true
    if (hasRuntime) onTunnelChanged((l) => (this.list = l))
    void this.reload()
  }

  async reload() {
    this.list = await backend.tunnels()
  }

  of(serverID: string, host: string, port: number): Tunnel | undefined {
    const h = host === '0.0.0.0' || host === '::' || host === '*' ? '127.0.0.1' : host
    return this.list.find(
      (t) => t.serverID === serverID && t.remotePort === port && t.remoteHost === h,
    )
  }

  async start(serverID: string, host: string, port: number, hostHeader: string) {
    const tun = await backend.start(serverID, host, port, hostHeader)
    if (!hasRuntime) await this.reload()
    return tun
  }

  async stop(id: string) {
    await backend.stop(id)
    if (!hasRuntime) await this.reload()
  }
}

export const tunnels = new Tunnels()
