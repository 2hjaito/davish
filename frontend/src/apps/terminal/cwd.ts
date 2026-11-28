// Where a terminal tab's shell is, from what the shell prints. This is
// server output: it only says which folder a button opens.

/** The path of an OSC 7 sequence ("file://host/path"), percent-decoded. */
export function parseOsc7(data: string): string | null {
  const path = /^file:\/\/[^/]*(\/.*)$/.exec(data)?.[1]
  if (!path) return null
  try {
    return decodeURIComponent(path)
  } catch {
    return path
  }
}

/** The folder in a "user@host: path" title, as bash sets it for xterm. */
export function cwdFromTitle(title: string): string | null {
  return /^[^\s:@]+@[^\s:]+:\s*((?:~|\/).*)$/.exec(title)?.[1] ?? null
}

/** An absolute path for what a prompt shows, or null when it cannot be one. */
export function absoluteCwd(cwd: string, home: string): string | null {
  if (cwd === '~') return home
  if (cwd.startsWith('~/')) return home + cwd.slice(1)
  return cwd.startsWith('/') ? cwd : null
}

/** The last folder of a path, what tells tabs apart when they get narrow: "~/a/projects" is "projects". */
export function lastFolder(cwd: string): string {
  if (cwd === '~' || cwd === '/') return cwd
  const parts = cwd.split('/').filter(Boolean)
  return parts[parts.length - 1] ?? cwd
}
