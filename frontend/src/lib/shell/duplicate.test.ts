import { describe, expect, it } from 'vitest'

import type { ServerInfo } from '../api/servers'
import { copyName, duplicateDraft } from './duplicate'

const server = (over: Partial<ServerInfo> = {}): ServerInfo => ({
  id: 'abc',
  name: 'app',
  host: '10.0.1.10',
  port: 2222,
  user: 'deploy',
  auth: 'key',
  keyPath: '~/.ssh/id_deploy',
  color: '#16a067',
  tag: 'local',
  group: 'my-amazing-app',
  jumpID: 'bastion',
  controlPath: '/run/user/1000/davish/app.sock',
  hasPassword: true,
  hasPassphrase: true,
  state: 'connected',
  ...over,
})

describe('copyName', () => {
  it('adds (copy) to the name', () => {
    expect(copyName('app', ['app'])).toBe('app (copy)')
  })

  it('counts up when the name is taken, ignoring case', () => {
    expect(copyName('app', ['app', 'App (Copy)'])).toBe('app (copy 2)')
    expect(copyName('app', ['app', 'app (copy)', 'app (copy 2)'])).toBe('app (copy 3)')
  })

  it('starts from the original name when copying a copy', () => {
    expect(copyName('app (copy)', ['app', 'app (copy)'])).toBe('app (copy 2)')
    expect(copyName('app (copy 2)', ['app', 'app (copy)', 'app (copy 2)'])).toBe('app (copy 3)')
  })

  it('keeps the result within the 64 characters the backend accepts', () => {
    const long = 'x'.repeat(64)
    const out = copyName(long, [long])
    expect(out.length).toBeLessThanOrEqual(64)
    expect(out.endsWith(' (copy)')).toBe(true)
    expect(copyName(long, [long, out]).length).toBeLessThanOrEqual(64)
  })
})

describe('duplicateDraft', () => {
  it('keeps how to reach the server and how it is listed', () => {
    const d = duplicateDraft(server(), ['app'])
    expect(d).toMatchObject({
      name: 'app (copy)',
      host: '10.0.1.10',
      port: 2222,
      user: 'deploy',
      auth: 'key',
      keyPath: '~/.ssh/id_deploy',
      color: '#16a067',
      tag: 'local',
      group: 'my-amazing-app',
      jumpID: 'bastion',
    })
  })

  it('has no id, no secret and no control socket', () => {
    const d = duplicateDraft(server(), ['app'])
    expect(d.id).toBe('')
    expect(d.password).toBe('')
    expect(d.passphrase).toBe('')
    expect(d.controlPath).toBe('')
  })

  it('fills the optional fields the way the dialog does', () => {
    const d = duplicateDraft(
      server({
        keyPath: undefined,
        color: undefined,
        tag: undefined,
        group: undefined,
        jumpID: undefined,
      }),
      [],
    )
    expect(d).toMatchObject({
      keyPath: '~/.ssh/id_ed25519',
      color: '',
      tag: '',
      group: '',
      jumpID: '',
    })
  })
})
