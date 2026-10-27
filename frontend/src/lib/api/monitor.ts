// Typed wrappers for the monitor bindings (internal/app/monitor.go).
import * as App from '../../../wailsjs/go/app/App'

/** Starts sysmon:tick for the server; pair every call with monitorUnwatch. */
export const monitorWatch = (serverID: string): Promise<void> => App.MonitorWatch(serverID)
export const monitorUnwatch = (serverID: string): Promise<void> => App.MonitorUnwatch(serverID)

export type ProcessSort = 'cpu' | 'mem'

export interface Process {
  pid: number
  ppid: number
  user: string
  /** The kernel's short name. */
  name: string
  /** Full command line. */
  command: string
  state: string
  /** Percent of one core, so it can pass 100 like in top. */
  cpu: number
  /** Resident bytes. */
  mem: number
  memPct: number
  threads: number
  /** Seconds since the process started. */
  age: number
}

export interface ProcessList {
  total: number
  procs: Process[]
}

/** The 50 busiest processes by sortBy; CPU is measured since the previous call. */
export const monitorProcesses = (serverID: string, sortBy: ProcessSort): Promise<ProcessList> =>
  App.MonitorProcesses(serverID, sortBy)

/** Fields the server does not reveal are empty. */
export interface SystemInfo {
  hostname: string
  os: string
  kernel: string
  arch: string
  cpuModel: string
  virt: string
}

export const monitorSystemInfo = (serverID: string): Promise<SystemInfo> =>
  App.MonitorSystemInfo(serverID)
