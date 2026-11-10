import { describe, expect, it } from 'vitest'
import { blockedIPs, deniedIPs } from './ufw'

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

describe('blockedIPs', () => {
  it('joins ufw and fail2ban, saying which blocks each address', () => {
    const got = blockedIPs(new Set(['10.0.0.9', '10.0.0.10']), new Set(['10.0.0.10', '2.2.2.2']))
    expect(got).toEqual([
      { ip: '2.2.2.2', via: ['fail2ban'] },
      { ip: '10.0.0.9', via: ['ufw'] },
      { ip: '10.0.0.10', via: ['ufw', 'fail2ban'] },
    ])
  })
})
