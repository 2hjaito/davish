// Typed wrappers for the disk bindings (internal/app/disk.go).
import * as App from '../../../wailsjs/go/app/App'

export interface Mount {
  path: string
  total: number
  used: number
}

export interface DiskEntry {
  path: string
  size: number
}

export interface Usage {
  dir: string
  size: number
  entries: DiskEntry[]
  /** du ran out of time; sizes are what it got to. */
  partial: boolean
}

export interface Cleanup {
  id: string
  bytes: number
  count?: number
  command: string
  /** Typed with sudo into a terminal instead of run. */
  root: boolean
}

export const diskMounts = (serverID: string): Promise<Mount[]> => App.DiskMounts(serverID)
export const diskUsage = (serverID: string, dir: string, token: string): Promise<Usage> =>
  App.DiskUsage(serverID, dir, token)
export const diskCancel = (token: string): Promise<void> => App.DiskCancel(token)
export const diskCleanups = (serverID: string): Promise<Cleanup[]> => App.DiskCleanups(serverID)
export const diskClean = (serverID: string, id: string, command: string): Promise<void> =>
  App.DiskClean(serverID, id, command)
