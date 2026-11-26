// The color schemes of Terminal (Settings → Terminal → Color scheme).
// Davish is the app's own look and is read from tokens.css, over the glass or
// a solid background; the others bring a background of their own, so they
// always draw solid. The colors are the schemes' published ones, artwork
// rather than tokens: a scheme must look the same in light and dark.
import type { ITheme } from '@xterm/xterm'

import type { TerminalScheme } from '../../lib/stores/settings.svelte'

export interface Scheme {
  background: string
  foreground: string
  cursor: string
  selection: string
  /** The 16 ANSI colors: black to white, then their bright versions. */
  ansi: readonly [
    string,
    string,
    string,
    string,
    string,
    string,
    string,
    string,
    string,
    string,
    string,
    string,
    string,
    string,
    string,
    string,
  ]
}

/** The schemes that bring their own colors. */
export const SCHEMES: Record<Exclude<TerminalScheme, 'davish'>, Scheme> = {
  phosphor: {
    background: '#050805',
    foreground: '#35e06b',
    cursor: '#5dff8f',
    selection: 'rgba(53, 224, 107, 0.25)',
    ansi: [
      '#050805',
      '#e5534b',
      '#35e06b',
      '#c8d64a',
      '#3fa7d6',
      '#b968c7',
      '#2ee6c5',
      '#bfe8c9',
      '#2c4a34',
      '#ff7b72',
      '#5dff8f',
      '#e6f06a',
      '#6cc4ee',
      '#d68ae3',
      '#6ff5dc',
      '#e8fff0',
    ],
  },
  solarized: {
    background: '#002b36',
    foreground: '#839496',
    cursor: '#93a1a1',
    selection: 'rgba(147, 161, 161, 0.25)',
    ansi: [
      '#073642',
      '#dc322f',
      '#859900',
      '#b58900',
      '#268bd2',
      '#d33682',
      '#2aa198',
      '#eee8d5',
      '#002b36',
      '#cb4b16',
      '#586e75',
      '#657b83',
      '#839496',
      '#6c71c4',
      '#93a1a1',
      '#fdf6e3',
    ],
  },
  dracula: {
    background: '#282a36',
    foreground: '#f8f8f2',
    cursor: '#f8f8f2',
    selection: 'rgba(98, 114, 164, 0.45)',
    ansi: [
      '#21222c',
      '#ff5555',
      '#50fa7b',
      '#f1fa8c',
      '#bd93f9',
      '#ff79c6',
      '#8be9fd',
      '#f8f8f2',
      '#6272a4',
      '#ff6e6e',
      '#69ff94',
      '#ffffa5',
      '#d6acff',
      '#ff92df',
      '#a4ffff',
      '#ffffff',
    ],
  },
}

/**
 * The ANSI colors of the Davish look. xterm's own defaults are dull: bright
 * black is #666, which almost vanishes on the dark glass, and that is the
 * grey that stack traces, timestamps and hints are printed in. These stay
 * readable on --term-bg-solid and on the glass above it. Green, blue and
 * yellow come from the tokens (--term-user, --term-path, --term-warn).
 */
export const DAVISH_ANSI = {
  black: '#6f7788',
  red: '#ff7a85',
  magenta: '#d6a2ff',
  cyan: '#7ee0e6',
  white: '#d7dae0',
  brightBlack: '#a3abbb',
  brightRed: '#ff9aa3',
  brightMagenta: '#e3bbff',
  brightCyan: '#9aeff4',
  brightWhite: '#ffffff',
} as const

/** The scheme's colors; undefined for Davish, whose colors come from tokens.css. */
export function schemeFor(id: TerminalScheme): Scheme | undefined {
  return id === 'davish' ? undefined : SCHEMES[id]
}

/** xterm's theme for a scheme, drawn on its own background or, when not solid, on the glass. */
export function schemeTheme(s: Scheme, solid: boolean): ITheme {
  const [black, red, green, yellow, blue, magenta, cyan, white] = s.ansi
  const [bBlack, bRed, bGreen, bYellow, bBlue, bMagenta, bCyan, bWhite] = s.ansi.slice(8)
  return {
    background: solid ? s.background : '#00000000',
    foreground: s.foreground,
    cursor: s.cursor,
    cursorAccent: s.background,
    selectionBackground: s.selection,
    black,
    red,
    green,
    yellow,
    blue,
    magenta,
    cyan,
    white,
    brightBlack: bBlack,
    brightRed: bRed,
    brightGreen: bGreen,
    brightYellow: bYellow,
    brightBlue: bBlue,
    brightMagenta: bMagenta,
    brightCyan: bCyan,
    brightWhite: bWhite,
  }
}

/** What a tile in Settings shows: the background, the text and three accents. */
export function swatch(id: TerminalScheme): { bg: string; fg: string; accents: string[] } {
  const s = schemeFor(id)
  if (!s) return { bg: '#1d1f25', fg: '#f2f3f5', accents: ['#7ee2a8', '#8ab4ff', '#ffb86b'] }
  return { bg: s.background, fg: s.foreground, accents: [s.ansi[2], s.ansi[4], s.ansi[3]] }
}
