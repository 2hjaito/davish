// Typed wrappers for the terminal bindings (internal/app/terminal.go).
import * as App from '../../../wailsjs/go/app/App'

import { decodeBase64 } from './base64'

/**
 * prompt asks for Davish's powerline prompt; it applies when the login shell
 * is bash. dir, when set, is where the shell starts (home if it cannot).
 */
export const termStart = (
  serverID: string,
  cols: number,
  rows: number,
  prompt: boolean,
  dir = '',
): Promise<string> => App.TermStart(serverID, cols, rows, prompt, dir)
export const termWrite = (id: string, data: string): Promise<void> => App.TermWrite(id, data)
export const termResize = (id: string, cols: number, rows: number): Promise<void> =>
  App.TermResize(id, cols, rows)
export const termClose = (id: string): Promise<void> => App.TermClose(id)

/** Decodes a term:data payload; output may not be valid UTF-8, so it stays bytes. */
export const decodeOutput = decodeBase64
