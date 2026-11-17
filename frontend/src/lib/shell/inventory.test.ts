import { describe, expect, it } from 'vitest'

import { address, groupCounts, issueText, mockInventory, visibleHosts } from './inventory'

describe('inventory import helpers', () => {
  const inv = mockInventory()

  it('filters hosts by group, through child groups too', () => {
    expect(visibleHosts(inv.hosts, '').length).toBe(inv.hosts.length)
    expect(visibleHosts(inv.hosts, 'web').map((h) => h.name)).toEqual([
      'web1.example.com',
      'web2.example.com',
      'web3.example.com',
    ])
    expect(visibleHosts(inv.hosts, 'prod').length).toBe(4)
  })

  it('counts only hosts not saved yet', () => {
    const c = groupCounts(inv.hosts)
    expect(c.get('web')).toBe(2)
    expect(c.get('prod')).toBe(3)
    expect(c.get('ungrouped')).toBe(1)
  })

  it('writes the address like ssh', () => {
    expect(address({ user: 'ops', address: '10.0.1.5', port: 2222 })).toBe('ops@10.0.1.5:2222')
    expect(address({ user: 'a', address: 'h', port: 22 })).toBe('a@h')
  })

  it('names the variable or value in a reason', () => {
    expect(issueText({ code: 'connection', detail: 'winrm' })).toContain('winrm')
    expect(issueText({ code: 'vault', detail: 'ansible_host' })).toContain('ansible_host')
    expect(issueText({ code: 'password' })).not.toBe('')
  })
})
