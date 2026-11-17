// Helpers shared by the server searches of Files and Logs.

/** Splits text around the first case-insensitive (unless matchCase) match, to highlight it. */
export function highlight(
  text: string,
  query: string,
  matchCase: boolean,
): { before: string; match: string; after: string } {
  const q = query.trim()
  const i = matchCase
    ? text.indexOf(q)
    : text.toLocaleLowerCase('en').indexOf(q.toLocaleLowerCase('en'))
  if (!q || i < 0) return { before: text, match: '', after: '' }
  return {
    before: text.slice(0, i),
    match: text.slice(i, i + q.length),
    after: text.slice(i + q.length),
  }
}

/** A unique name for a search, for its Stop button. */
export function searchToken(): string {
  return `search-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`
}
