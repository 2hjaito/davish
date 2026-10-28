// Typed wrappers for the file bindings (internal/app/files.go).
import * as App from '../../../wailsjs/go/app/App'

import { decodeBase64, encodeBase64 } from './base64'

export type EntryKind = 'file' | 'dir' | 'link' | 'other'

export interface Entry {
  name: string
  /** Absolute and clean, as every path the file bindings accept. */
  path: string
  kind: EntryKind
  /** True for directories and links to one. */
  dir: boolean
  /** Where a link points, as stored. */
  target?: string
  size: number
  /** Like ls -l: "drwxr-xr-x". */
  mode: string
  perm: number
  /** Unix seconds. */
  modTime: number
  uid: number
  gid: number
}

export interface RemovePlan extends Entry {
  /** A directory with content: confirm twice and make the user type its name. */
  nonEmpty: boolean
  /** Exactly what filesRemove runs; show it in the confirmation dialog. */
  command: string
  /** The same command with sudo, for a terminal when the server refuses. */
  sudo: string
}

/** Content plus the stat it was read with; larger than 10 MiB fails with FS_TOO_LARGE. */
export interface FileContent extends Entry {
  data: Uint8Array
}

// Go sends []byte as base64; the generated types claim number[].
interface RawFile extends Entry {
  data: string
}

export const filesHome = (serverID: string): Promise<string> => App.FilesHome(serverID)

export const filesList = (serverID: string, dir: string): Promise<Entry[]> =>
  App.FilesList(serverID, dir) as Promise<Entry[]>

export const filesStat = (serverID: string, path: string): Promise<Entry> =>
  App.FilesStat(serverID, path) as Promise<Entry>

export async function filesRead(serverID: string, path: string): Promise<FileContent> {
  const raw = (await App.FilesRead(serverID, path)) as unknown as RawFile
  return { ...raw, data: decodeBase64(raw.data) }
}

/** Replaces the file atomically; ask the user first when it already exists. */
export const filesWrite = (serverID: string, path: string, data: Uint8Array): Promise<Entry> =>
  App.FilesWrite(serverID, path, encodeBase64(data) as unknown as number[]) as Promise<Entry>

export const filesMkdir = (serverID: string, path: string): Promise<void> =>
  App.FilesMkdir(serverID, path)

/** Fails with FS_EXISTS rather than replacing `to`. */
export const filesRename = (serverID: string, from: string, to: string): Promise<void> =>
  App.FilesRename(serverID, from, to)

export const filesPlanRemove = (serverID: string, path: string): Promise<RemovePlan> =>
  App.FilesPlanRemove(serverID, path) as Promise<RemovePlan>

/** Runs plan.command; fails with FS_STALE when the path changed since the plan. */
export const filesRemove = (serverID: string, plan: RemovePlan): Promise<void> =>
  App.FilesRemove(serverID, plan.path, plan.command)

/** zip, tar.gz (shown as GZ) or tar; folders and files alike. */
export type ArchiveFormat = 'zip' | 'tar.gz' | 'tar'

export interface CompressPlan {
  /** Next to the source, never an existing name. */
  output: string
  /** Exactly what filesCompress runs. */
  command: string
  /** The same archiver with sudo, for a terminal when the folder is not writable. */
  sudo: string
}

export const filesPlanCompress = (
  serverID: string,
  path: string,
  format: ArchiveFormat,
): Promise<CompressPlan> => App.FilesPlanCompress(serverID, path, format)

/** Runs plan.command; fails with FS_STALE when the plan no longer holds. */
export const filesCompress = (
  serverID: string,
  path: string,
  format: ArchiveFormat,
  plan: CompressPlan,
): Promise<CompressPlan> => App.FilesCompress(serverID, path, format, plan.command)

export interface SaveResult {
  entry: Entry
  /** Path of the copy made of the previous version, when asked for. */
  backup?: string
}

/**
 * Saves editor text atomically. With backup, the current file is first
 * copied to "<name>.bak-<UTC time>" next to it.
 */
export const filesSave = (
  serverID: string,
  path: string,
  data: Uint8Array,
  opts: {
    backup: boolean
    /** The file as it was opened; saving fails with FS_CHANGED if the server copy differs. */
    expect?: { modTime: number; size: number }
  },
): Promise<SaveResult> =>
  App.FilesSave(
    serverID,
    path,
    encodeBase64(data) as unknown as number[],
    // Plain data is what crosses the bridge; the generated class only adds helpers.
    opts as Parameters<typeof App.FilesSave>[3],
  ) as Promise<SaveResult>

export interface Access {
  /** Saving would work: file and folder writable, owner kept. */
  writable: boolean
  /** Command to prefill in a terminal to edit the file as root. */
  sudoEdit: string
}

export const filesAccess = (serverID: string, path: string): Promise<Access> =>
  App.FilesAccess(serverID, path)
