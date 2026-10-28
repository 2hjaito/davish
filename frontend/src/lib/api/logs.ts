// Typed wrappers for the log bindings (internal/app/logs.go).
import * as App from '../../../wailsjs/go/app/App'

export type LogKind = 'journal' | 'file' | 'docker'

/** A unit (empty for the whole journal), an absolute file path or a container name. */
export interface LogSource {
  kind: LogKind
  target: string
}

/** Starts following src; lines arrive through onLogsLine, the end through onLogsEnd. */
export const logsStart = (serverID: string, src: LogSource): Promise<string> =>
  App.LogsStart(serverID, src)

export const logsStop = (streamID: string): Promise<void> => App.LogsStop(streamID)
