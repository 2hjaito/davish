// The Git sidebar groups repositories by the account or org their origin
// remote belongs to: the one the server signs in as first, then the others
// by name, then those without a remote.
import type { Identity, Repo } from '../../lib/api/git'
import { mainLogin } from './account'

export interface RepoGroup {
  /** "octocat" on GitHub, "gitlab.com/team" elsewhere; empty without a remote. */
  key: string
  repos: Repo[]
}

/** A group's name: the owner, with its host unless it is GitHub. */
export function ownerKey(host: string, owner: string): string {
  if (!owner) return ''
  return host === 'github.com' ? owner : `${host}/${owner}`
}

/** The group of the account the server's keys sign in as, if any. */
export function myKey(id: Identity | null): string {
  const login = mainLogin(id)
  if (!login) return ''
  // A deploy key names owner/repo; the owner is the group.
  const owner = login.deploy ? login.account.split('/')[0] : login.account
  return ownerKey(login.host, owner ?? '')
}

const byName = (a: string, b: string) => a.localeCompare(b, undefined, { sensitivity: 'base' })

export function groupRepos(repos: readonly Repo[], mine: string): RepoGroup[] {
  const groups = new Map<string, RepoGroup>()
  for (const r of repos) {
    const key = ownerKey(r.host, r.owner)
    // Owners differ only in case on GitHub; one group holds them.
    const id = key.toLowerCase()
    const g = groups.get(id) ?? { key, repos: [] }
    g.repos.push(r)
    groups.set(id, g)
  }
  const me = mine.toLowerCase()
  const rank = (g: RepoGroup) => (g.key === '' ? 2 : me && g.key.toLowerCase() === me ? 0 : 1)
  return [...groups.values()]
    .map((g) => ({
      ...g,
      repos: [...g.repos].sort((a, b) => byName(a.name, b.name) || byName(a.path, b.path)),
    }))
    .sort((a, b) => rank(a) - rank(b) || byName(a.key, b.key))
}
