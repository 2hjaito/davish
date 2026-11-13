// Copying a selection out of the virtual log view. The rows on screen are
// absolutely placed divs, so the browser's own text of a selection can lose
// its line breaks; the copy is rebuilt from the lines themselves.

/** A point in the log: a line index and a character offset in it. */
export interface Point {
  row: number
  offset: number
}

const before = (a: Point, b: Point) => a.row < b.row || (a.row === b.row && a.offset <= b.offset)

/** The text between two points, either order, lines joined by newlines. */
export function sliceLines(lines: readonly string[], a: Point, b: Point): string {
  const [from, to] = before(a, b) ? [a, b] : [b, a]
  const out: string[] = []
  for (let r = Math.max(0, from.row); r <= Math.min(lines.length - 1, to.row); r++) {
    const line = lines[r] ?? ''
    const start = r === from.row ? from.offset : 0
    const end = r === to.row ? to.offset : line.length
    out.push(line.slice(start, end))
  }
  return out.join('\n')
}
