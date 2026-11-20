import { describe, expect, it, vi } from 'vitest'

import type { HostLogin, Identity } from '../../lib/api/git'

const gitlab = Promise.withResolvers<HostLogin>()
const backend = vi.hoisted(() => ({
  identity: vi.fn<(id: string) => Promise<Identity>>(),
  hostLogin: vi.fn<(id: string, host: string) => Promise<HostLogin>>(),
}))
vi.mock('./backend', () => ({ backend }))

const { identities } = await import('./identities.svelte')

describe('identities', () => {
  it('shows each host as it answers, and asks once per session', async () => {
    backend.identity.mockResolvedValue({
      user: 'root',
      home: '/root',
      host: 'web',
      gitName: '',
      gitEmail: '',
      helper: '',
      keys: [],
      hosts: [],
    })
    backend.hostLogin.mockImplementation((_id, host) =>
      host === 'github.com'
        ? Promise.resolve({ host, state: 'ok', account: 'octocat', deploy: false })
        : gitlab.promise,
    )

    const done = identities.load('s1')
    await vi.waitFor(() => expect(identities.get('s1').identity?.hosts).toHaveLength(1))
    expect(identities.get('s1').pending).toEqual(['gitlab.com'])

    gitlab.resolve({ host: 'gitlab.com', state: 'unreachable', account: '', deploy: false })
    await done
    expect(identities.get('s1').pending).toEqual([])

    await identities.load('s1')
    expect(backend.identity).toHaveBeenCalledTimes(1)
    await identities.load('s1', true)
    expect(backend.identity).toHaveBeenCalledTimes(2)
  })
})
