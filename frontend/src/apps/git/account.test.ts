import { describe, expect, it } from 'vitest'

import type { Identity } from '../../lib/api/git'
import { accountName, accountPicture, mainLogin, sshReadiness } from './account'

const base: Identity = {
  user: 'root',
  home: '/root',
  host: 'web',
  gitName: '',
  gitEmail: '',
  helper: '',
  keys: [],
  hosts: [],
}

describe('account', () => {
  it('prefers a user login over a deploy key', () => {
    const id: Identity = {
      ...base,
      hosts: [
        { host: 'github.com', state: 'ok', account: 'o/app', deploy: true },
        { host: 'gitlab.com', state: 'ok', account: 'dang', deploy: false },
      ],
    }
    expect(mainLogin(id)?.account).toBe('dang')
    expect(accountName(id)).toBe('dang')
    expect(accountPicture(id)).toBe('')
  })

  it('falls back to git name, then the SSH user', () => {
    expect(accountName({ ...base, gitName: 'Dang Tran' })).toBe('Dang Tran')
    expect(accountName(base)).toBe('root')
    expect(accountName(null)).toBe('')
  })

  it('asks GitHub for the picture of a GitHub login', () => {
    const id: Identity = {
      ...base,
      hosts: [{ host: 'github.com', state: 'ok', account: 'octocat', deploy: false }],
    }
    expect(accountPicture(id)).toBe('https://github.com/octocat.png?size=96')
    expect(sshReadiness(id, 'github.com')?.state).toBe('ok')
    expect(sshReadiness(id, 'git.example.com')).toBeNull()
  })
})
