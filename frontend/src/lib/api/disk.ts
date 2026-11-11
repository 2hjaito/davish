// Typed wrappers for the disk bindings (internal/app/disk.go).
import * as App from '../../../wailsjs/go/app/App'

import type { RemovePlan } from './files'

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
  /** Needs root: runs as sudoCommand through sudo (ADR 0018), or in a terminal. */
  root: boolean
  sudoCommand?: string
  /** May delete data, not only caches and leftovers. */
  danger?: boolean
}

export const diskMounts = (serverID: string): Promise<Mount[]> => App.DiskMounts(serverID)
/** A deleted file a process still holds open (internal/disk.DeletedFile). */
export interface DeletedFile {
  path: string
  size: number
  pid: number
  command: string
}

/** Where a filesystem's space goes (internal/disk.Overview). */
export interface Overview {
  mount: Mount
  measured: number
  /** Folders down to three levels below the mount, largest first. */
  dirs: DiskEntry[]
  /** The largest files, largest first. */
  files: DiskEntry[]
  /** Folders du could not enter; their size is in unmeasured. */
  unreadable: string[]
  deleted: DeletedFile[]
  /** Used space neither du nor deleted account for. */
  unmeasured: number
  /** Measured as root through sudo -n. */
  root: boolean
  partial: boolean
  /** Read-only, typed into a terminal: measures as root with the user's password. */
  sudoMeasure: string
  /** sudo could measure it all once given the session's password. */
  askSudo: boolean
  /** Exactly what measuring as root runs, for the sudo prompt. */
  sudoScan?: string
}

/** How a file would be emptied (internal/disk.TruncatePlan). */
export interface TruncatePlan {
  command: string
  sudo: string
}

export const diskOverview = (serverID: string, mount: string, token: string): Promise<Overview> =>
  App.DiskOverview(serverID, mount, token) as Promise<Overview>
export const diskPlanRemove = (serverID: string, path: string): Promise<RemovePlan> =>
  App.DiskPlanRemove(serverID, path) as Promise<RemovePlan>
export const diskRemove = (serverID: string, path: string, command: string): Promise<void> =>
  App.DiskRemove(serverID, path, command)
export const diskPlanTruncate = (path: string): Promise<TruncatePlan> => App.DiskPlanTruncate(path)
export const diskTruncate = (serverID: string, path: string, command: string): Promise<void> =>
  App.DiskTruncate(serverID, path, command)
export const diskUsage = (serverID: string, dir: string, token: string): Promise<Usage> =>
  App.DiskUsage(serverID, dir, token)
export const diskCancel = (token: string): Promise<void> => App.DiskCancel(token)
export const diskCleanups = (serverID: string): Promise<Cleanup[]> => App.DiskCleanups(serverID)
export const diskClean = (serverID: string, id: string, command: string): Promise<void> =>
  App.DiskClean(serverID, id, command)
