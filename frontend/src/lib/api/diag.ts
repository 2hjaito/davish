// Typed wrappers for the diagnosis bindings (internal/app/diag.go).
import * as App from '../../../wailsjs/go/app/App'

export type Verdict =
  'ok' | 'no-connection' | 'shell-hangs' | 'shell-slow' | 'ss-hangs' | 'docker-hangs'

export interface DiagStep {
  id: string
  ok: boolean
  detail?: string
  ms: number
}

export interface DiagReport {
  verdict: Verdict
  steps: DiagStep[]
  findings?: { file: string; line: number; text: string }[]
  /** Plain text of the report, for a support message. */
  text: string
}

export interface SupportInfo {
  version: string
  os: string
  logPath: string
  logTail: string
}

export const diagRun = (serverID: string): Promise<DiagReport> =>
  App.DiagRun(serverID) as Promise<DiagReport>
export const diagSupportInfo = (): Promise<SupportInfo> => App.DiagSupportInfo()
