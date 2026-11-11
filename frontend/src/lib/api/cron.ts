// Typed wrappers for the cron bindings (internal/app/cron.go).
import * as App from '../../../wailsjs/go/app/App'

/** A schedule expanded to the values each field allows; dow 0 is Sunday. */
export interface Schedule {
  raw: string
  reboot: boolean
  minute: number[]
  hour: number[]
  dom: number[]
  month: number[]
  dow: number[]
  /** The field was "*": with both day fields restricted, either one matching runs. */
  domAny: boolean
  dowAny: boolean
}

export interface CronJob {
  /** Names the line and its content; an edit of a changed line is refused. */
  id: string
  /** "user", "root" (root's crontab, through sudo), "/etc/crontab" or "/etc/cron.d/<name>". */
  source: string
  line: number
  schedule: Schedule
  /** System crontabs only. */
  user?: string
  command: string
  enabled: boolean
  /** Unix seconds, up to three; empty for @reboot. */
  next: number[]
  /** The user's own crontab, and root's through sudo. */
  editable: boolean
}

export interface CronTable {
  jobs: CronJob[]
  env: { source: string; name: string; value: string }[]
  /** The server's IANA zone, in which next runs are computed. */
  zone: string
  unreadable: string[]
  /** The user's crontab as read. */
  version: string
  /** Root's crontab is listed and editable through sudo. */
  root: boolean
  rootVersion: string
  /** sudo could show root's crontab and the unreadable files once given the password. */
  rootLocked: boolean
}

export type CronOp = 'add' | 'edit' | 'delete' | 'enable' | 'disable'

export interface CronChange {
  /** "root" for root's crontab; the user's own otherwise. */
  source: 'user' | 'root'
  op: CronOp
  id: string
  schedule: string
  command: string
}

export interface CronPlan {
  /** Exactly what runs; the new crontab goes in on stdin. */
  command: string
  /** command as it runs through sudo, for root's crontab; empty otherwise. */
  sudo: string
  before: string
  after: string
  /** Where the current crontab is saved first, if there is one. */
  backup: string
  version: string
}

export interface CronPreview {
  schedule: Schedule
  next: number[]
}

export interface CronRuns {
  lines: string[]
  restricted: boolean
  sudo: string
  askSudo: boolean
}

export const cronList = (serverID: string): Promise<CronTable> =>
  App.CronList(serverID) as Promise<CronTable>

export const cronPlan = (serverID: string, change: CronChange): Promise<CronPlan> =>
  App.CronPlan(serverID, change)

/** Makes change exactly as plan was confirmed; refused if the crontab changed since. */
export const cronApply = (serverID: string, change: CronChange, plan: CronPlan): Promise<void> =>
  App.CronApply(serverID, change, plan)

export const cronPreview = (schedule: string, zone: string): Promise<CronPreview> =>
  App.CronPreview(schedule, zone) as Promise<CronPreview>

export const cronRuns = (serverID: string, command: string): Promise<CronRuns> =>
  App.CronRuns(serverID, command)
