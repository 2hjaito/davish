// Searching the server from Files: scopes, demo results and how hits are
// shown. The search itself runs on the server (internal/fs/search.go).
export { highlight, searchToken } from '../../lib/search'
import type { SearchHit, SearchMode, SearchResult } from '../../lib/api/files'
import { highlight } from '../../lib/search'
import { baseName, parentPath } from './listing'
import { mockRead, mockWalk } from './mockFs'

/** Where the search box looks: the listing on screen, or the server. */
export type Scope = 'here' | 'tree' | 'content'

export const SCOPES: readonly Scope[] = ['here', 'tree', 'content']

export function modeOf(scope: Scope): SearchMode | null {
  if (scope === 'tree') return 'name'
  if (scope === 'content') return 'content'
  return null
}

/** Content search cannot start where grep would read devices or kernel files. */
export function contentAllowed(dir: string): boolean {
  if (dir === '/') return false
  return !['/proc', '/sys', '/dev', '/run'].some((v) => dir === v || dir.startsWith(v + '/'))
}

/** A hit's name and the folder that holds it, relative to where the search started. */
export function describe(hit: SearchHit, from: string): { name: string; where: string } {
  const parent = parentPath(hit.path)
  let where = parent
  if (parent === from) where = '.'
  else if (from !== '/' && parent.startsWith(from + '/')) where = parent.slice(from.length + 1)
  return { name: baseName(hit.path), where }
}

function globToRegExp(glob: string, matchCase: boolean): RegExp {
  const src = glob
    .replace(/[.+^${}()|\\]/g, '\\$&')
    .replace(/\*/g, '.*')
    .replace(/\?/g, '.')
  return new RegExp(`^${src}$`, matchCase ? '' : 'i')
}

/** What the server would answer, on the browser preview's demo tree. */
export function mockSearch(
  dir: string,
  query: string,
  mode: SearchMode,
  matchCase: boolean,
): SearchResult {
  const q = query.trim()
  const hits: SearchHit[] = []
  if (mode === 'name') {
    const re = globToRegExp(/[*?[]/.test(q) ? q : `*${q}*`, matchCase)
    for (const e of mockWalk(dir)) if (re.test(baseName(e.path))) hits.push({ path: e.path })
  } else {
    for (const e of mockWalk(dir)) {
      if (e.dir) continue
      mockRead(e.path)
        .split('\n')
        .forEach((text, i) => {
          if (highlight(text, q, matchCase).match)
            hits.push({ path: e.path, line: i + 1, text: text.trim() })
        })
    }
  }
  const command =
    mode === 'name'
      ? `find ${dir} -mindepth 1 -iname '*${q}*' -print0`
      : `grep -rIn -i -e '${q}' -- ${dir}`
  return {
    hits: hits.slice(0, 500),
    truncated: hits.length > 500,
    denied: 0,
    timedOut: false,
    command,
  }
}
