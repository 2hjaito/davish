/** The part of an xterm buffer belowContent reads. */
export interface BufferLike {
  readonly length: number
  /** Index of the first line in the viewport. */
  readonly baseY: number
  getLine(y: number): { translateToString(trimRight?: boolean): string } | undefined
}

/**
 * Escape sequence that puts the cursor at the start of the last line with
 * text, so a notice written after it, then "\r\n", lands below everything.
 *
 * A program that redraws its output in place (docker build progress, npm,
 * top) leaves the cursor somewhere in the middle of the screen when the
 * connection drops. Writing there overwrites old lines without clearing
 * them, and the next shell's prompt piles on top of them.
 *
 * Expects scroll margins and origin mode reset (DECSTR), as CUP is relative
 * to the viewport.
 */
export function belowContent(buf: BufferLike): string {
  let last = buf.length - 1
  while (last >= buf.baseY && !buf.getLine(last)?.translateToString(true)) last--
  // Nothing in the viewport: start at its top.
  if (last < buf.baseY) return '\x1b[H'
  return `\x1b[${last - buf.baseY + 1};1H`
}
