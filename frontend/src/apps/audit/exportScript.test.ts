import { describe, expect, it } from 'vitest'

import type { AuditEntry } from '../../lib/api/events'
import { changes, defaultPick, mockScript, plain } from './exportScript'

const e = (id: number, over: Partial<AuditEntry>): AuditEntry => ({
  id,
  time: Date.UTC(2026, 10, 16, 8, 0, id),
  serverID: 'web',
  app: 'services',
  command: 'uptime',
  exit: 0,
  durMs: 10,
  ...over,
})

describe('Command log export', () => {
  const list = [
    e(1, {}),
    e(2, { mutating: true, command: 'systemctl restart nginx' }),
    e(3, { mutating: true, exit: 1, command: 'docker rm web' }),
    e(4, { mutating: true, input: true, command: 'tee /etc/x >/dev/null' }),
  ]

  it('keeps only changes, and ticks the ones that succeeded', () => {
    expect(changes(list).map((x) => x.id)).toEqual([2, 3, 4])
    expect(defaultPick(list)).toEqual([2, 4])
  })

  it('shows sudo as a person types it', () => {
    expect(plain("LC_ALL=C sudo -n -- sh -c 'ls /root'")).toBe("sudo sh -c 'ls /root'")
    expect(plain("LC_ALL=C sudo -S -k -p '' -- sh -c 'ls'")).toBe("sudo sh -c 'ls'")
    expect(plain('ls')).toBe('ls')
  })

  it('writes a preview script like Go does', () => {
    const s = mockScript(changes(list), () => ({ name: 'web', ssh: 'ssh -t deploy@web' }), false)
    expect(s.startsWith('#!/usr/bin/env bash\n')).toBe(true)
    expect(s).toContain("ssh -t deploy@web -- 'systemctl restart nginx'")
    expect(s).toContain('#   tee /etc/x >/dev/null')
    expect(s).not.toContain('docker rm web')
    expect(mockScript(changes(list), () => ({ name: 'web', ssh: 'ssh' }), true)).toContain(
      'exit 1 then',
    )
  })
})
