import { describe, expect, it } from 'vitest'

import type { Identity, Repo } from '../../lib/api/git'
import { groupRepos, myKey, ownerKey } from './groups'

const repo = (name: string, host = '', owner = ''): Repo => ({
  path: `/var/www/${name}`,
  name,
  host,
  owner,
})

describe('groupRepos', () => {
  it('puts my account first, the others by name, no remote last', () => {
    const groups = groupRepos(
      [
        repo('site'),
        repo('web', 'github.com', 'tsc'),
        repo('api', 'github.com', 'TSC'),
        repo('bio', 'github.com', 'theptaydo'),
        repo('ci', 'gitlab.com', 'team/sub'),
      ],
      'theptaydo',
    )
    expect(groups.map((g) => g.key)).toEqual(['theptaydo', 'gitlab.com/team/sub', 'tsc', ''])
    expect(groups[2]?.repos.map((r) => r.name)).toEqual(['api', 'web'])
  })

  it('names the group of a login or a deploy key', () => {
    const id = (account: string, deploy: boolean, host = 'github.com'): Identity => ({
      user: 'root',
      home: '/root',
      host: 'web',
      gitName: '',
      gitEmail: '',
      helper: '',
      keys: [],
      hosts: [{ host, state: 'ok', account, deploy }],
    })
    expect(myKey(id('octocat', false))).toBe('octocat')
    expect(myKey(id('acme/site', true))).toBe('acme')
    expect(myKey(id('dang', false, 'gitlab.com'))).toBe('gitlab.com/dang')
    expect(myKey(null)).toBe('')
    expect(ownerKey('github.com', '')).toBe('')
  })
})
