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
  /** "user", "/etc/crontab" or "/etc/cron.d/<name>". */
  source: string
  line: number
  schedule: Schedule
  /** System crontabs only. */
  user?: string
  command: string
  enabled: boolean
  /** Unix seconds, up to three; empty for @reboot. */
  next: number[]
  /** Only the user's own crontab. */
  editable: boolean
}

export interface CronTable {
  jobs: CronJob[]
  env: { source: string; name: string; value: string }[]
  /** The server's IANA zone, in which next runs are computed. */
  zone: string
  unreadable: string[]
  /** The user's crontab as read, for cronApply. */
  version: string
}

export type CronOp = 'add' | 'edit' | 'delete' | 'enable' | 'disable'

export interface CronChange {
  op: CronOp
  id: string
  schedule: string
  command: string
}

export interface CronPlan {
  /** Exactly what runs; the new crontab goes in on stdin. */
  command: string
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
}

export const cronList = (serverID: string): Promise<CronTable> =>
  App.CronList(serverID) as Promise<CronTable>

export const cronPlan = (serverID: string, change: CronChange): Promise<CronPlan> =>
  App.CronPlan(serverID, change)

export const cronApply = (serverID: string, change: CronChange, version: string): Promise<void> =>
  App.CronApply(serverID, change, version)

export const cronPreview = (schedule: string, zone: string): Promise<CronPreview> =>
  App.CronPreview(schedule, zone) as Promise<CronPreview>

export const cronRuns = (serverID: string, command: string): Promise<CronRuns> =>
  App.CronRuns(serverID, command)
