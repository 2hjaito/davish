// Typed wrapper for the audit binding (internal/app/audit.go).
import * as App from '../../../wailsjs/go/app/App'

import type { AuditEntry } from './events'

/** Commands run so far, oldest first; new ones come through onAuditEntry. */
export const auditEntries = (): Promise<AuditEntry[]> => App.AuditEntries()
