// Typed wrappers for the port and tunnel bindings (internal/app/ports.go).
import * as App from '../../../wailsjs/go/app/App'

/** One listening TCP socket on the server. */
export interface Port {
  addr: string
  port: number
  /** Empty when the socket is another user's and Davish is not root. */
  process?: string
  pid?: number
  /** The container docker-proxy publishes the port for. */
  container?: string
  loopback: boolean
}

export interface PortList {
  ports: Port[]
  root: boolean
}

/** A server port forwarded to 127.0.0.1:localPort on this machine. */
export interface Tunnel {
  id: string
  serverID: string
  remoteHost: string
  remotePort: number
  localPort: number
  /** The port was taken here, so another was used. */
  moved: boolean
  hostHeader?: string
  conns: number
}

export const portsList = (serverID: string): Promise<PortList> => App.PortsList(serverID)

export const portsProbe = (
  serverID: string,
  host: string,
  port: number,
  hostHeader: string,
): Promise<string> => App.PortsProbe(serverID, host, port, hostHeader)

export const tunnelStart = (
  serverID: string,
  host: string,
  port: number,
  hostHeader: string,
): Promise<Tunnel> => App.TunnelStart(serverID, host, port, hostHeader)

export const tunnelStop = (id: string): Promise<void> => App.TunnelStop(id)
export const tunnelList = (): Promise<Tunnel[]> => App.TunnelList()
