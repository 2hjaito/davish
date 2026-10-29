// Demo ports for the browser preview, like a small VPS running a web app.
import type { PortList, Tunnel } from '../../lib/api/ports'

const delay = <T>(v: T, ms = 150) => new Promise<T>((r) => setTimeout(() => r(v), ms))

export function mockList(): Promise<PortList> {
  return delay({
    root: false,
    ports: [
      { addr: '0.0.0.0', port: 22, process: 'sshd', pid: 812, loopback: false, exposed: 'yes' },
      { addr: '127.0.0.53', port: 53, loopback: true, exposed: 'no' },
      { addr: '0.0.0.0', port: 80, process: 'nginx', pid: 1201, loopback: false, exposed: 'yes' },
      { addr: '0.0.0.0', port: 443, process: 'nginx', pid: 1201, loopback: false, exposed: 'yes' },
      { addr: '127.0.0.1', port: 3000, process: 'node', pid: 2210, loopback: true, exposed: 'no' },
      { addr: '127.0.0.1', port: 3306, loopback: true, exposed: 'no' },
      {
        addr: '127.0.0.1',
        port: 6379,
        process: 'redis-server',
        pid: 990,
        loopback: true,
        exposed: 'no',
      },
      {
        addr: '0.0.0.0',
        port: 8080,
        process: 'docker-proxy',
        pid: 3100,
        container: 'web',
        loopback: false,
        exposed: 'yes',
      },
      {
        addr: '0.0.0.0',
        port: 6380,
        process: 'redis-server',
        pid: 991,
        loopback: false,
        exposed: 'yes',
      },
    ],
    firewall: { state: 'active', defaultAllow: false },
  })
}

let tunnels: Tunnel[] = []
let next = 1

export function mockTunnels(): Promise<Tunnel[]> {
  return delay([...tunnels])
}

export function mockStart(serverID: string, host: string, port: number, hostHeader: string) {
  const tun: Tunnel = {
    id: `t${next++}`,
    serverID,
    remoteHost: host === '0.0.0.0' ? '127.0.0.1' : host,
    remotePort: port,
    localPort: port === 3306 ? 53306 : port,
    moved: port === 3306,
    hostHeader: hostHeader || undefined,
    conns: 0,
  }
  tunnels = [...tunnels, tun]
  return delay(tun)
}

export function mockStop(id: string): Promise<void> {
  tunnels = tunnels.filter((t) => t.id !== id)
  return delay(undefined)
}

export function mockProbe(_serverID: string, _host: string, port: number, hostHeader: string) {
  return delay(
    `HTTP/1.1 200 OK\r\nServer: nginx\r\nContent-Type: text/html\r\n\r\n<!doctype html><title>${hostHeader || `port ${port}`}</title>`,
    400,
  )
}
