// A compose project's icon or framework, read once per server and folder for
// the session: the folder's files rarely change, and each read is an SFTP
// round trip or a few.
import type { StackIcon } from '../../lib/api/docker'
import { type Tech, techIcons } from '../../lib/ui/techIcons'
import { backend } from './backend'

const cache = new Map<string, Promise<StackIcon>>()

export function projectIcon(serverID: string, dir: string): Promise<StackIcon> {
  const key = `${serverID}\n${dir}`
  let p = cache.get(key)
  if (!p) {
    // An unreadable folder or a dropped connection just means no icon.
    p = backend.stackIcon(serverID, dir).catch(() => ({}))
    cache.set(key, p)
  }
  return p
}

/** The tech the server named, when this build has its logo. */
export function knownTech(tech: string | undefined): Tech | undefined {
  return tech && tech in techIcons ? (tech as Tech) : undefined
}
