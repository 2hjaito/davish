// Typed wrappers for the log bindings (internal/app/logs.go).
import * as App from '../../../wailsjs/go/app/App'

export type LogKind = 'journal' | 'file' | 'docker'

/** A unit (empty for the whole journal), an absolute file path or a container name. */
export interface LogSource {
  kind: LogKind
  target: string
}

/** A started stream, and the same command with sudo for an account that cannot read it. */
export interface LogsStarted {
  streamID: string
  sudo: string
}

/** Starts following src; lines arrive through onLogsLine, the end through onLogsEnd. */
export const logsStart = (serverID: string, src: LogSource): Promise<LogsStarted> =>
  App.LogsStart(serverID, src)

/** The same as root through sudo; never asks for the password itself (ADR 0018). */
export const logsStartAsRoot = (serverID: string, src: LogSource): Promise<LogsStarted> =>
  App.LogsStartAsRoot(serverID, src)

export const logsStop = (streamID: string): Promise<void> => App.LogsStop(streamID)

export interface LogSearchQuery {
  query: string
  matchCase: boolean
}

/** A line with its number in the file; 0 for journal and container lines. */
export interface LogLine {
  number: number
  text: string
}

export interface LogSearchResult {
  /** The latest matches, oldest first. */
  lines: LogLine[]
  /** 500 lines came back: older matches may exist. */
  truncated: boolean
  timedOut: boolean
  /** The journal hid other users' entries. */
  restricted: boolean
  command: string
}

export interface LogPage {
  lines: LogLine[]
  /** The file ended before the page did. */
  eof: boolean
}

/** Searches src's whole history on the server; token names it for logsCancelSearch. */
export const logsSearch = (
  serverID: string,
  src: LogSource,
  q: LogSearchQuery,
  token: string,
): Promise<LogSearchResult> => App.LogsSearch(serverID, src, q, token)

export const logsCancelSearch = (token: string): Promise<void> => App.LogsCancelSearch(token)

/** count (≤ 200) lines of a log file from 1-based line from. */
export const logsAround = (
  serverID: string,
  file: string,
  from: number,
  count: number,
): Promise<LogPage> => App.LogsAround(serverID, file, from, count)
