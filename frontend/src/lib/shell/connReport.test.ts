import { describe, expect, it } from 'vitest'

import type { Server } from '../stores/servers.svelte'
import { connReport, serverLogLines } from './connReport'

const tail = [
  '{"time":"t1","level":"INFO","msg":"connected","server":"a1"}',
  '{"time":"t2","level":"INFO","msg":"ran command","server":"a1","cmd":"cat /srv/secret-project/x"}',
  '{"time":"t3","level":"WARN","msg":"connection state","server":"a1","state":"failed","err":"INTERNAL"}',
  '{"time":"t4","level":"WARN","msg":"connection state","server":"b2","state":"failed"}',
  '',
].join('\n')

const srv: Server = {
  id: 'a1',
  name: 'package',
  host: 'vps.dangth.dev',
  port: 22,
  user: 'dangth',
  auth: 'key',
  keyPath: '~/.ssh/id_ed25519',
  hasPassword: false,
  hasPassphrase: true,
  state: 'failed',
  error: 'INTERNAL',
  errorMessage: "cannot read Davish's known_hosts",
  errorDetail: 'load known_hosts: knownhosts: ~/x:3: illegal base64 data',
} as Server

describe('serverLogLines', () => {
  it('keeps only connection lines of that server', () => {
    const lines = serverLogLines(tail, 'a1')
    expect(lines).toHaveLength(2)
    expect(lines.join()).not.toContain('secret-project')
  })
})

describe('connReport', () => {
  const text = connReport(
    { version: '1.13.7', os: 'windows/amd64', logPath: '', logTail: tail },
    srv,
  )

  it('names the version, auth, code, message and cause', () => {
    expect(text).toContain('Davish 1.13.7 · windows/amd64')
    expect(text).toContain('Auth: private key, passphrase saved')
    expect(text).toContain('Code: INTERNAL')
    expect(text).toContain("Message: cannot read Davish's known_hosts")
    expect(text).toContain('Detail: load known_hosts')
  })

  it('leaves out the host, user and key path', () => {
    expect(text).not.toContain('vps.dangth.dev')
    expect(text).not.toContain('id_ed25519')
  })
})
