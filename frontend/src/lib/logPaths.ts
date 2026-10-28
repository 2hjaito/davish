// Finds absolute file paths in a log line, so a click can open the file in
// the Editor at the line the log names.

export interface Segment {
  text: string
  /** Set when text is a path, without the line suffix. */
  path?: string
  /** 1-based line named next to the path. */
  line?: number
}

// An absolute path of at least two parts, not part of a URL or a longer
// word, with an optional ":12" or ":12:5" after it.
const PATH = /(?<![\w:/.~-])(\/(?:[\w.@+-]+\/)+[\w.@+-]*[\w@+-])(?::(\d+)(?::\d+)?)?/g
// Top folders of a Linux tree. A path under another one counts only with
// a file extension, so "GET /api/orders" in a web log is not a link.
const ROOTS = new Set([
  'app',
  'bin',
  'boot',
  'data',
  'etc',
  'home',
  'lib',
  'lib64',
  'mnt',
  'opt',
  'root',
  'run',
  'sbin',
  'srv',
  'tmp',
  'usr',
  'var',
  'www',
])
const EXT = /\.[A-Za-z0-9]{1,8}$/

function looksLikeFile(path: string): boolean {
  const top = path.slice(1, path.indexOf('/', 1))
  return ROOTS.has(top) || EXT.test(path)
}

// Python's traceback: File "/app/x.py", line 42
const PY_LINE = /^",\s*line\s+(\d+)/

/** Splits line into plain text and path segments, in order. */
export function findPaths(line: string): Segment[] {
  const out: Segment[] = []
  let last = 0
  for (const m of line.matchAll(PATH)) {
    const start = m.index
    const path = m[1]!
    if (!looksLikeFile(path)) continue
    const end = start + m[0].length
    let n = m[2] ? Number(m[2]) : undefined
    if (n === undefined) {
      const py = PY_LINE.exec(line.slice(end))
      if (py) n = Number(py[1])
    }
    if (start > last) out.push({ text: line.slice(last, start) })
    out.push({ text: line.slice(start, end), path, line: n && n > 0 ? n : undefined })
    last = end
  }
  if (last < line.length) out.push({ text: line.slice(last) })
  return out
}
