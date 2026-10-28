// How a log line reads at a glance, from the words in it. Shared by every
// view of server logs (Services, Docker, Logs).

export type LineLevel = 'error' | 'warn' | null

const ERROR = /\b(error|err|failed|failure|fatal|crit(ical)?|emerg|alert|panic|segfault)\b/i
const WARN = /\b(warn(ing)?|deprecated)\b/i

export function lineLevel(line: string): LineLevel {
  if (ERROR.test(line)) return 'error'
  if (WARN.test(line)) return 'warn'
  return null
}
