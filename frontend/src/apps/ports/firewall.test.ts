import { describe, expect, it } from 'vitest'
import type { Port } from '../../lib/api/ports'
import { closeAllCommand, exposedAdmin } from './firewall'

const p = (x: Partial<Port>): Port => ({
  addr: '0.0.0.0',
  port: 0,
  loopback: false,
  exposed: 'yes',
  ...x,
})

describe('exposedAdmin', () => {
  it('splits ufw-closable ports from containers', () => {
    const r = exposedAdmin([
      p({ port: 6379, process: 'redis-server' }),
      p({ port: 6379, addr: '::', process: 'redis-server' }),
      p({ port: 5432, exposed: 'no' }),
      p({ port: 80, process: 'nginx' }),
      p({ port: 3000, process: 'docker-proxy', container: 'grafana' }),
    ])
    expect(r.ufw.map((x) => x.port)).toEqual([6379])
    expect(r.docker.map((x) => x.container)).toEqual(['grafana'])
    expect(closeAllCommand(r.ufw)).toBe(
      'sudo ufw delete allow 6379/tcp ; sudo ufw delete allow 6379',
    )
  })
})
