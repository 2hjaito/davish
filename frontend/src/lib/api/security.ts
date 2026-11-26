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
  /** sudo runs without asking for a password. */
  sudo: boolean
  /** Actions can run through sudo, with the session's password asked once (ADR 0018). */
  canSudo: boolean
  /** What needs root was read as root. */
  asRoot: boolean
  /** Root would show more once given the password; sudoRead is what then runs. */
  askSudo: boolean
  sudoRead?: string
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
  /** Davish's own drop-in turned it off, so Davish can undo it (ADR 0029). */
  passwordOurs: boolean
  disablePassword: string
}

/** What the Security app can do to one address (internal/security/actions.go). */
export type SecurityAction = 'block' | 'unblock' | 'ban' | 'unban'

export interface SecurityPlan {
  /** What Davish runs once confirmed: as root, or with sudo -n. */
  command: string
  /** The same with plain sudo, for Terminal. */
  terminal: string
}

export const securityReport = (serverID: string): Promise<SecurityReport> =>
  App.SecurityReport(serverID)

export const securityPlan = (
  action: SecurityAction,
  ip: string,
  root: boolean,
): Promise<SecurityPlan> => App.SecurityPlan(action, ip, root)

export const securityApply = (
  serverID: string,
  action: SecurityAction,
  ip: string,
  command: string,
): Promise<void> => App.SecurityApply(serverID, action, ip, command)

/** What sshd does about password logins, and why turning them off is not offered. */
export interface PasswordStatus {
  password: string
  kbdInteractive: string
  ours: boolean
  /** noInclude, match, methods, control or password (internal/security/password.go). */
  blocked: string[]
}

export interface PasswordPlan {
  status: SecurityPlan
  disable: SecurityPlan
  /** Undoes disable: removes the drop-in and reloads sshd. */
  enable: SecurityPlan
}

export const securityPasswordStatus = (serverID: string): Promise<PasswordStatus> =>
  App.SecurityPasswordStatus(serverID)

export const securityPasswordPlan = (root: boolean): Promise<PasswordPlan> =>
  App.SecurityPasswordPlan(root)

export const securityDisablePassword = (serverID: string, command: string): Promise<void> =>
  App.SecurityDisablePassword(serverID, command)

export const securityEnablePassword = (serverID: string, command: string): Promise<void> =>
  App.SecurityEnablePassword(serverID, command)
