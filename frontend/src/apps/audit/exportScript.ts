// Choosing Command log entries to export as a script (.claude/docs/adr/0022).
import type { AuditEntry } from '../../lib/api/events'

/** Entries that change a server: what the export starts from. */
export const changes = (entries: readonly AuditEntry[]): AuditEntry[] =>
  entries.filter((e) => e.mutating)

/** What is ticked when export starts: every shown change that succeeded. */
export const defaultPick = (shown: readonly AuditEntry[]): number[] =>
  shown.filter((e) => e.mutating && e.exit === 0).map((e) => e.id)

// Davish's sudo forms (internal/sudo), shown as a plain sudo.
const SUDO = [/^LC_ALL=C sudo -n -- sh -c /, /^LC_ALL=C sudo -S -k -p '' -- sh -c /]

/** The command as a person types it, like Go's audit.Plain. */
export function plain(command: string): string {
  for (const re of SUDO) if (re.test(command)) return command.replace(re, 'sudo sh -c ')
  return command
}

const quote = (s: string) => `'${s.replaceAll("'", `'\\''`)}'`

/**
 * The script for the browser preview, which has no Go side; the real one
 * comes from audit.Script and has the same shape.
 */
export function mockScript(
  entries: readonly AuditEntry[],
  target: (serverID: string) => { name: string; ssh: string },
  failed: boolean,
): string {
  const lines = [
    '#!/usr/bin/env bash',
    `# Exported from the Davish Command log at ${new Date().toISOString().slice(0, 16).replace('T', ' ')} UTC.`,
    "# Each command runs over its own ssh connection, in the server's login",
    '# shell (Davish ran them with sh -c). Read it before you run it.',
    'set -euo pipefail',
  ]
  let server = ''
  for (const e of [...entries].sort((a, b) => a.id - b.id)) {
    if (e.exit !== 0 && !failed) continue
    const t = target(e.serverID)
    if (e.serverID !== server) {
      server = e.serverID
      lines.push('', `# --- ${t.name} ---`)
    }
    const when = new Date(e.time).toISOString().slice(0, 19).replace('T', ' ')
    lines.push('', `# ${when} ${e.app}${e.exit !== 0 ? `, exit ${e.exit} then` : ''}`)
    if (e.input) {
      lines.push('# content sent on stdin was not recorded; the command was:')
      lines.push(
        ...plain(e.command)
          .split('\n')
          .map((l) => `#   ${l}`),
      )
    } else lines.push(`${t.ssh} -- ${quote(plain(e.command))}`)
  }
  return lines.join('\n') + '\n'
}
