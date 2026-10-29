import { describe, expect, it } from 'vitest'
import type { Port } from '../../lib/api/ports'
import { choices } from './upstreams'

const port = (p: Partial<Port>): Port =>
  ({ addr: '0.0.0.0', port: 0, loopback: false, ...p }) as Port

describe('choices', () => {
  it('lists what listens, without nginx and sshd, once per address', () => {
    expect(
      choices([
        port({ port: 80, process: 'nginx' }),
        port({ port: 22, process: 'sshd' }),
        port({ port: 3000, addr: '127.0.0.1', loopback: true, process: 'node' }),
        port({ port: 3000, addr: '::1', loopback: true, process: 'node' }),
        port({ port: 8080, container: 'web' }),
        port({ port: 5000, addr: '10.0.0.2' }),
      ]),
    ).toEqual([
      { value: '10.0.0.2:5000', label: '' },
      { value: '127.0.0.1:3000', label: 'node' },
      { value: '127.0.0.1:8080', label: 'web' },
    ])
  })
})
