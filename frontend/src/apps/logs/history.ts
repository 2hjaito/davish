// Paging a log file around a search hit, and the browser preview's
// stand-ins for the server searches (internal/logs/search.go).
import type { LogLine, LogPage, LogSearchResult } from '../../lib/api/logs'
import { highlight } from '../../lib/search'

/** Lines per page when reading around a hit; the server allows 200. */
export const PAGE = 100

/** The first line of the page that puts line n in its middle. */
export function aroundStart(n: number): number {
  return Math.max(1, n - PAGE / 2)
}

/** Searches the lines on screen, as the server would search the whole file. */
export function mockLogSearch(
  lines: readonly string[],
  query: string,
  matchCase: boolean,
): LogSearchResult {
  const hits: LogLine[] = []
  lines.forEach((text, i) => {
    if (highlight(text, query, matchCase).match) hits.push({ number: i + 1, text })
  })
  return {
    lines: hits.slice(-500),
    truncated: hits.length >= 500,
    timedOut: false,
    restricted: false,
    command: `grep -F -i -n -e '${query}' -- /var/log/syslog | tail -n 500`,
  }
}

export function mockAround(lines: readonly string[], from: number, count: number): LogPage {
  const page = lines
    .slice(from - 1, from - 1 + count)
    .map((text, i) => ({ number: from + i, text }))
  return { lines: page, eof: from - 1 + count >= lines.length }
}
