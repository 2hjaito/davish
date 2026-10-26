export type ConnState = 'connected' | 'connecting' | 'disconnected'

export interface Server {
  id: string
  name: string
  user: string
  os?: string
  state: ConnState
}

// Mock data until conn.Manager lands in M2.
const mockServers: Server[] = [
  { id: 'tsc-vps', name: 'tsc-vps', user: 'dangth', os: 'Ubuntu', state: 'connected' },
  { id: 'nas', name: 'NAS văn phòng', user: 'dangth', state: 'disconnected' },
]

class Servers {
  list = $state<Server[]>(mockServers)
  activeId = $state(mockServers[0]?.id ?? '')

  active = $derived(this.list.find((s) => s.id === this.activeId))

  select(id: string) {
    if (this.list.some((s) => s.id === id)) this.activeId = id
  }

  selectIndex(i: number) {
    const s = this.list[i]
    if (s) this.activeId = s.id
  }
}

export const servers = new Servers()
