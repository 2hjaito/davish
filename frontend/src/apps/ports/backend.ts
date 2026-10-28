// The port and tunnel calls, or demo data in the plain browser preview.
import { portsList, portsProbe, tunnelList, tunnelStart, tunnelStop } from '../../lib/api/ports'
import { hasRuntime } from '../../lib/api/runtime'
import { mockList, mockProbe, mockStart, mockStop, mockTunnels } from './mock'

export const backend = hasRuntime
  ? {
      list: portsList,
      probe: portsProbe,
      start: tunnelStart,
      stop: tunnelStop,
      tunnels: tunnelList,
    }
  : {
      list: (_serverID: string) => mockList(),
      probe: mockProbe,
      start: mockStart,
      stop: mockStop,
      tunnels: mockTunnels,
    }
