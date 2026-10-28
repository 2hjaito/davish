// Typed wrappers for the git bindings (internal/app/git.go).
import * as App from '../../../wailsjs/go/app/App'

export interface Repo {
  path: string
  name: string
}

export interface Commit {
  hash: string
  short: string
  author: string
  /** Unix seconds. */
  time: number
  subject: string
}

/**
 * A file that differs from the last commit. index and worktree are git's
 * status letters ("M", "A", "D", "R", "C", "T"), "." when unchanged on that
 * side, "?" when untracked.
 */
export interface Change {
  path: string
  origPath: string
  index: string
  worktree: string
  conflict: boolean
}

export interface Status {
  /** Empty on a detached HEAD. */
  branch: string
  /** Empty when the branch tracks nothing. */
  upstream: string
  ahead: number
  behind: number
  /** Null in a repository without commits. */
  last: Commit | null
  changes: Change[]
  truncated: boolean
}

/** A commit when commit is set, else file in the working tree against HEAD. */
export interface DiffRequest {
  dir: string
  commit: string
  file: string
  untracked: boolean
}

export interface Diff {
  text: string
  truncated: boolean
}

/** How many commits gitLog returns at most (git.LogPage). */
export const LOG_PAGE = 100

export const gitFind = (serverID: string, extra: string[]): Promise<Repo[]> =>
  App.GitFind(serverID, extra)
export const gitStatus = (serverID: string, dir: string): Promise<Status> =>
  App.GitStatus(serverID, dir) as Promise<Status>
export const gitLog = (serverID: string, dir: string, skip: number): Promise<Commit[]> =>
  App.GitLog(serverID, dir, skip)
export const gitDiff = (serverID: string, req: DiffRequest): Promise<Diff> =>
  App.GitDiff(serverID, req)
