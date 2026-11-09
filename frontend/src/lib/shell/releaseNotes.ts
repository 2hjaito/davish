/** Where release.yml starts the Vietnamese notes, folded under the English ones. */
const VI_MARKER = '<!-- davish:vi -->'

/**
 * The notes of a release in the interface's language: Vietnamese for vi,
 * English for every other language. Releases from before the split carry
 * one language only and come back whole.
 */
export function releaseNotes(body: string, lang: string): string {
  const at = body.indexOf(VI_MARKER)
  if (at < 0) return body
  if (lang !== 'vi') return body.slice(0, at)
  const vi = body.slice(at + VI_MARKER.length)
  const end = vi.indexOf('</details>')
  return (end < 0 ? vi : vi.slice(0, end)).replace(/<details>|<summary>.*?<\/summary>/g, '').trim()
}
