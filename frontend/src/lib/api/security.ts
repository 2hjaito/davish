// Typed wrappers for the security binding (internal/app/security.go).
import * as App from '../../../wailsjs/go/app/App'

export interface Login {
  time: string
  user: string
  ip: string
  method: string
}

export interface Attempts {
  ip: string
  count: number
  users: string[]
  last: string
}

export interface Firewall {
  installed: boolean
  needsRoot: boolean
  active: boolean
  lines: string[]
  banned?: string[]
}

export interface SecurityReport {
  root: boolean
  /** 'journal', 'auth.log', or '' when neither was readable. */
  source: string
  accepted: Login[]
  failed: Attempts[]
  failedTotal: number
  last: string[]
  lastb: string[]
  lastbDenied: boolean
  ufw: Firewall
  fail2ban: Firewall
  /** sshd's PasswordAuthentication: 'yes', 'no' or '' when unknown. */
  passwordAuth: string
  disablePassword: string
}

export const securityReport = (serverID: string): Promise<SecurityReport> =>
  App.SecurityReport(serverID)
