// Pure helpers of saved commands: which show on a server, search, and a
// warning for commands that look like they hold a secret.

export interface Snippet {
  id: string
  name: string
  command: string
  /** The server it belongs to, '' for every server. */
  serverID: string
}

/** The snippets of serverID and those for every server, by name. */
export function visibleOn(list: readonly Snippet[], serverID: string, query = ''): Snippet[] {
  const q = query.trim().toLowerCase()
  return list
    .filter((s) => s.serverID === '' || s.serverID === serverID)
    .filter((s) => !q || `${s.name}\n${s.command}`.toLowerCase().includes(q))
    .sort((a, b) => a.name.localeCompare(b.name))
}

// Saved commands sit in settings.json in plain text; a secret must not
// (docs/SECURITY.md). These are the usual shapes of one on a command line.
const SECRET = [
  /\b(pass(word)?|passwd|pwd|secret|token|api[-_]?key|access[-_]?key)\s*[=:]\s*\S+/i,
  /\s-p\S+/, // mysql -psecret
  /\bAuthorization:\s*\S+/i,
  /:\/\/[^\s/:@]+:[^\s/@]+@/, // scheme://user:pass@host
  /-----BEGIN [A-Z ]*PRIVATE KEY-----/,
  /\b(ghp|gho|github_pat|glpat|xox[bap]|sk-[a-z]*)[-_][A-Za-z0-9_-]{10,}/,
]

/** True when command looks like it carries a password, token or key. */
export function looksSecret(command: string): boolean {
  return SECRET.some((re) => re.test(command))
}

/** A short random id, unique enough for one user's list. */
export function newId(): string {
  return Math.random().toString(36).slice(2, 10) + Date.now().toString(36)
}
