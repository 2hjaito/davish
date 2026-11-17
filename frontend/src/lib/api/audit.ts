// Typed wrapper for the audit binding (internal/app/audit.go).
import * as App from '../../../wailsjs/go/app/App'

import type { AuditEntry } from './events'

/** Commands run so far, oldest first; new ones come through onAuditEntry. */
export const auditEntries = (): Promise<AuditEntry[]> => App.AuditEntries()

/** A bash script that runs the chosen entries again over ssh (docs/adr/0022). */
export const auditScript = (ids: number[], failed: boolean): Promise<string> =>
  App.AuditScript(ids, failed)

/** Saves that script where the user picks; '' when canceled. */
export const auditSaveScript = (ids: number[], failed: boolean, title: string): Promise<string> =>
  App.AuditSaveScript(ids, failed, title)
