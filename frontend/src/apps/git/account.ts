// What the Git account card and page show of a server's identity.
import type { HostLogin, Identity } from '../../lib/api/git'

/** The code host login worth showing first: a user account over a deploy key. */
export function mainLogin(id: Identity | null): HostLogin | null {
  const ok = (id?.hosts ?? []).filter((h) => h.state === 'ok')
  return ok.find((h) => !h.deploy) ?? ok[0] ?? null
}

/** The name the card shows: the code host account, git's name, or the SSH user. */
export function accountName(id: Identity | null): string {
  const login = mainLogin(id)
  if (login && !login.deploy) return login.account
  return id?.gitName || id?.user || ''
}

/** A GitHub account's picture; other hosts and deploy keys have none to ask for. */
export function accountPicture(id: Identity | null): string {
  const login = mainLogin(id)
  if (!login || login.deploy || login.host !== 'github.com') return ''
  return `https://github.com/${encodeURIComponent(login.account)}.png?size=96`
}

/** How a clone from host over SSH will go, from what host answered. */
export function sshReadiness(id: Identity | null, host: string): HostLogin | null {
  return id?.hosts.find((h) => h.host === host) ?? null
}
