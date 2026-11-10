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

export interface HourCount {
  /** Server time, "2006-01-02T15". */
  hour: string
  count: number
}

export interface UserCount {
  user: string
  count: number
}

export interface Session {
  user: string
  tty: string
  since: string
  /** The remote host, '' for a local login. */
  from: string
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
  /** Failed logins for a user that does not exist. */
  invalidTotal: number
  /** Addresses that failed, before failed is cut to 50. */
  failedIPs: number
  acceptedTotal: number
  targets: UserCount[]
  /** Failed logins per hour up to now, gaps filled, at most a week. */
  hourly: HourCount[] | null
  sessions: Session[] | null
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
