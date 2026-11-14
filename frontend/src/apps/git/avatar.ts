// Author avatars: the GitHub picture of a noreply address, else the
// Gravatar of the email, over initials drawn from the name that stay
// when no picture loads or the machine is offline.

/** Number of --graph-N colors an avatar can take. */
export const AVATAR_COLORS = 7

/** Up to two letters: first and last word, or the first two of one word. */
export function initials(name: string): string {
  const words = name
    .trim()
    .split(/[\s._-]+/)
    .filter(Boolean)
  const first = words[0]
  if (!first) return '?'
  const last = words[words.length - 1] ?? first
  const letters =
    words.length > 1 ? [...first].slice(0, 1).concat([...last].slice(0, 1)) : [...first].slice(0, 2)
  return letters.join('').toUpperCase()
}

/** A stable color index for a name, so one author keeps one color. */
export function avatarColor(name: string): number {
  let h = 0
  for (const ch of name.trim().toLowerCase()) h = (h * 31 + (ch.codePointAt(0) ?? 0)) >>> 0
  return h % AVATAR_COLORS
}

const SIZE = 40 // twice the 20px circle, for sharp pictures on Retina

/** The GitHub picture of a noreply address, which carries the account. */
export function githubAvatar(email: string): string | null {
  const m = /^(?:(\d+)\+)?([^@\s]+)@users\.noreply\.github\.com$/i.exec(email.trim())
  if (!m) return null
  const [, id, login] = m
  if (id) return `https://avatars.githubusercontent.com/u/${id}?s=${SIZE}&v=4`
  return `https://github.com/${encodeURIComponent(login ?? '')}.png?size=${SIZE}`
}

/** The Gravatar of an email; d=404 makes a missing one fail, so initials stay. */
export async function gravatar(email: string): Promise<string | null> {
  const norm = email.trim().toLowerCase()
  if (!norm.includes('@') || !globalThis.crypto?.subtle) return null
  const digest = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(norm))
  const hex = [...new Uint8Array(digest)].map((b) => b.toString(16).padStart(2, '0')).join('')
  return `https://gravatar.com/avatar/${hex}?s=${SIZE}&d=404`
}

const candidates = new Map<string, Promise<string[]>>()
const failed = new Set<string>()

/** The pictures to try for an email, best first; asked once per email. */
export function avatarUrls(email: string): Promise<string[]> {
  const key = email.trim().toLowerCase()
  let urls = candidates.get(key)
  if (!urls) {
    const gh = githubAvatar(key)
    urls = gravatar(key)
      .catch(() => null)
      .then((g) => [gh, g].filter((u): u is string => u !== null))
    candidates.set(key, urls)
  }
  return urls.then((list) => list.filter((u) => !failed.has(u)))
}

/** Remembers a picture that did not load, so other rows skip it. */
export function markFailed(url: string) {
  failed.add(url)
}
