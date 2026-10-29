import { describe, expect, it } from 'vitest'
import { blockCommand, deniedIPs } from './ufw'

describe('deniedIPs', () => {
  it('reads the addresses of deny and reject rules above every allow', () => {
    const lines = [
      'Status: active',
      'To                         Action      From',
      'Anywhere                   DENY        193.24.211.219',
      'Anywhere                   REJECT      10.0.0.0/8',
      'Anywhere (v6)              DENY        2001:db8::1',
      '22/tcp                     ALLOW       Anywhere',
      'Anywhere                   DENY        171.253.171.195',
    ]
    expect([...deniedIPs(lines)]).toEqual(['193.24.211.219', '10.0.0.0/8', '2001:db8::1'])
  })
})

describe('blockCommand', () => {
  it('replaces an older rule with one at the top', () => {
    expect(blockCommand('1.2.3.4')).toBe(
      'sudo ufw delete deny from 1.2.3.4 ; sudo ufw insert 1 deny from 1.2.3.4',
    )
  })
})
