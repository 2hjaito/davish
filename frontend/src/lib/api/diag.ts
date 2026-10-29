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

let reported = 0

/**
 * Sends errors the interface hits to the app's log file, so a support copy
 * shows them. At most 50 per run, in case one repeats on every frame.
 */
export function reportFrontendErrors(hasRuntime: boolean) {
  if (!hasRuntime) return
  const send = (what: unknown) => {
    if (reported++ >= 50) return
    const text =
      what instanceof Error ? `${what.name}: ${what.message}\n${what.stack ?? ''}` : String(what)
    void App.DiagFrontendError(text).catch(() => {})
  }
  window.addEventListener('error', (ev) => send(ev.error ?? ev.message))
  window.addEventListener('unhandledrejection', (ev) => send(ev.reason))
}
