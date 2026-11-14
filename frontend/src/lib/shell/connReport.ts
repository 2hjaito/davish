import type { SupportInfo } from '../api/diag'
import type { Server } from '../stores/servers.svelte'

// Connection events only: other lines of the server may name its files.
const connMsgs = [
  'connection state',
  'connected',
  'connection lost',
  'auth failed',
  'host key changed, refusing',
]

/** The connection's log lines for one server, newest last. Server IDs are random, not names. */
export function serverLogLines(logTail: string, serverID: string, max = 12): string[] {
  return logTail
    .split('\n')
    .filter(
      (l) =>
        l.includes(`"server":"${serverID}"`) && connMsgs.some((m) => l.includes(`"msg":"${m}"`)),
    )
    .slice(-max)
}

/**
 * Plain text a user pastes into a bug report about a failed connection. It
 * names the auth method but never the host, user, key path or a secret.
 */
export function connReport(info: SupportInfo | null, srv: Server): string {
  const auth =
    srv.auth === 'key'
      ? `private key${srv.hasPassphrase ? ', passphrase saved' : ''}`
      : srv.auth === 'password'
        ? 'password'
        : 'automatic (agent, ~/.ssh keys)'
  const lines = info ? serverLogLines(info.logTail, srv.id) : []
  return [
    info ? `Davish ${info.version} · ${info.os}` : '',
    `Auth: ${auth}${srv.jumpID ? ' · via jump host' : ''}`,
    `Code: ${srv.error ?? 'INTERNAL'}`,
    srv.errorMessage ? `Message: ${srv.errorMessage}` : '',
    srv.errorDetail ? `Detail: ${srv.errorDetail}` : '',
    lines.length ? `\nLog:\n${lines.join('\n')}` : '',
  ]
    .filter(Boolean)
    .join('\n')
}
