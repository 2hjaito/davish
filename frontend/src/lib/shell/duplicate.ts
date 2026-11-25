// Duplicating a server: a new entry that starts as a copy of one, opened in the
// add dialog so only what differs is typed. Pure; the dialog does the saving.
import type { ServerDraft, ServerInfo } from '../api/servers'

/** Longest server name the backend accepts (internal/store). */
const MAX_NAME = 64

const COPY_SUFFIX = / \(copy(?: \d+)?\)$/

/**
 * "name (copy)", or "name (copy 2)", "name (copy 3)"… the first one no server
 * has yet, ignoring case. Copying a copy starts from the original's name, and
 * a long name is cut so the result still fits.
 */
export function copyName(name: string, taken: readonly string[]): string {
  const base = name.trim().replace(COPY_SUFFIX, '')
  const used = new Set(taken.map((n) => n.trim().toLowerCase()))
  for (let n = 1; ; n++) {
    const suffix = n === 1 ? ' (copy)' : ` (copy ${n})`
    const candidate = base.slice(0, MAX_NAME - suffix.length).trimEnd() + suffix
    if (!used.has(candidate.toLowerCase())) return candidate
  }
}

/**
 * The add dialog's starting point for a copy of `source`: everything that
 * describes how to reach it and how it is listed, but no id and no secret, and
 * not its control socket (a socket belongs to one master of one server).
 */
export function duplicateDraft(source: ServerInfo, taken: readonly string[]): ServerDraft {
  return {
    id: '',
    name: copyName(source.name, taken),
    host: source.host,
    port: source.port,
    user: source.user,
    auth: source.auth,
    keyPath: source.keyPath ?? '~/.ssh/id_ed25519',
    color: source.color ?? '',
    tag: source.tag ?? '',
    group: source.group ?? '',
    jumpID: source.jumpID ?? '',
    controlPath: '',
    password: '',
    passphrase: '',
  }
}
