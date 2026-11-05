// The files and folders a cron command mentions. Only for showing links to
// them: nothing here is ever run, and each path is checked on the server.

const INTERPRETERS = new Set(
  'sh bash zsh dash python python2 python3 php node ruby perl flock nice ionice timeout env sudo'.split(
    ' ',
  ),
)
const SYSTEM = /^\/(bin|sbin|usr\/bin|usr\/sbin|dev|proc|sys)(\/|$)/
const SCRIPT = /\.(sh|bash|py|php|js|mjs|rb|pl)$/
const OPERATORS = /^(&&|\|\||[;|&]|\d*>>?|<|\d*>&\d*)$/
export const MAX_PATHS = 6

/** Words and operators of a command, quotes removed. */
function tokenize(command: string): string[] {
  const out: string[] = []
  let word = ''
  let started = false
  let quote = ''
  const flush = () => {
    if (started) out.push(word)
    word = ''
    started = false
  }
  for (let i = 0; i < command.length; i++) {
    const c = command[i]!
    if (quote) {
      if (c === quote) quote = ''
      else word += c
    } else if (c === '"' || c === "'") {
      quote = c
      started = true
    } else if (c === '\\') {
      word += command[++i] ?? ''
      started = true
    } else if (/\s/.test(c)) {
      flush()
    } else if (';|&<>'.includes(c)) {
      // A digit before > is the stream, as in 2>&1.
      const stream = /^\d+$/.test(word) && c === '>' ? word : ''
      if (!stream) flush()
      word = ''
      started = false
      let op = stream + c
      while (';|&<>'.includes(command[i + 1] ?? ' ') || /\d/.test(command[i + 1] ?? ' ')) {
        if (/\d/.test(command[i + 1]!) && !op.endsWith('&')) break
        op += command[++i]
      }
      out.push(op)
    } else {
      word += c
      started = true
    }
  }
  flush()
  return out
}

function normalize(path: string): string {
  const parts: string[] = []
  for (const p of path.split('/')) {
    if (p === '..') parts.pop()
    else if (p && p !== '.') parts.push(p)
  }
  return '/' + parts.join('/')
}

/** A word as an absolute path, when it reads as one. */
function pathOf(word: string, cwd: string | undefined, home: string | undefined): string | null {
  if (!word || word.startsWith('-') || /[$`*?]/.test(word) || word.includes('://')) return null
  if (word.includes('=') && !word.startsWith('/')) return null
  if (word.startsWith('/')) return normalize(word)
  if (word === '~' || word.startsWith('~/')) return home ? normalize(home + word.slice(1)) : null
  if (!cwd || (!word.includes('/') && !SCRIPT.test(word))) return null
  return normalize(`${cwd}/${word}`)
}

/**
 * The paths in a crontab command, in order, without programs and system
 * folders. Cron starts jobs in the home folder, and a cd moves that along;
 * home is empty when the job's user is someone else.
 */
export function commandPaths(command: string, home = ''): string[] {
  // In a crontab, the first unescaped % ends the command.
  const end = /(^|[^\\])%/.exec(command)
  const line = end ? command.slice(0, end.index + end[1]!.length) : command

  const found: string[] = []
  let cwd = home || undefined
  let first = true
  let afterCd = false
  for (const word of tokenize(line)) {
    if (OPERATORS.test(word)) {
      first = !/>|</.test(word)
      afterCd = false
      continue
    }
    if (afterCd) {
      afterCd = false
      const dir = pathOf(word, cwd, home)
      cwd = dir ?? undefined
      if (dir && dir !== '/') found.push(dir)
      continue
    }
    if (first && /^\w+=/.test(word)) continue
    const program = first
    first = false
    if (program && word === 'cd') {
      afterCd = true
      continue
    }
    if (INTERPRETERS.has(word.split('/').pop() ?? '')) {
      first = true
      continue
    }
    const path = pathOf(word, cwd, home)
    if (path && !SYSTEM.test(path)) found.push(path)
  }
  return [...new Set(found)].slice(0, MAX_PATHS)
}

/** The folder holding path. */
export function folderOf(path: string): string {
  return path.slice(0, path.lastIndexOf('/')) || '/'
}
