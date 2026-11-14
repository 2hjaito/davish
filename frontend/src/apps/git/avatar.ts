// Author avatars drawn from the name alone: no email leaves the machine
// and no image is fetched.

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
