import type { SupportInfo } from '../api/diag'

/** Log lines worth sending about an update: the updater's own, newest last. */
export function updateLogLines(logTail: string, max = 20): string[] {
  return logTail
    .split('\n')
    .filter((l) => l.includes('"msg":"update'))
    .slice(-max)
}

/** Plain text for a bug report on a failed update. Logs never hold secrets. */
export function updateReport(info: SupportInfo | null, error: string): string {
  const lines = info ? updateLogLines(info.logTail) : []
  return [
    info ? `Davish ${info.version} · ${info.os}` : '',
    `Error: ${error}`,
    lines.length ? `\nLog (${info?.logPath ?? ''}):\n${lines.join('\n')}` : '',
  ]
    .filter(Boolean)
    .join('\n')
}
