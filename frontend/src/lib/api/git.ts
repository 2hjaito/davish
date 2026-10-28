// Typed wrappers for the git bindings (internal/app/git.go).
import * as App from '../../../wailsjs/go/app/App'

export interface Repo {
  path: string
  name: string
}

export type RefKind = 'head' | 'local' | 'remote' | 'tag'

/** A branch or tag pointing at a commit; 'head' is the one checked out. */
export interface Ref {
  name: string
  kind: RefKind
}

export interface Commit {
  hash: string
  short: string
  author: string
  /** Unix seconds. */
  time: number
  subject: string
  /** Filled by gitLog only, to draw the graph. */
  parents: string[]
  refs: Ref[]
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
  /** Installs Node.js dependencies as the lockfile pins them; empty when none. */
  install: string
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

export type GitOp = 'fetch' | 'pull' | 'switch'

/** The exact command of an operation, to confirm. */
export interface Plan {
  command: string
  /** Tracked files with uncommitted changes; a pull is refused unless 0. */
  dirty: number
}

/** What git printed; error is an apperr code, empty on success. */
export interface OpResult {
  output: string
  error: string
}

export interface Branch {
  name: string
  /** Only on a remote: switching creates the local branch. */
  remote: boolean
  current: boolean
}

/** How many commits gitLog returns at most (git.LogPage). */
export const LOG_PAGE = 100

export const gitFind = (serverID: string, extra: string[]): Promise<Repo[]> =>
  App.GitFind(serverID, extra)
export const gitStatus = (serverID: string, dir: string): Promise<Status> =>
  App.GitStatus(serverID, dir) as Promise<Status>
export const gitLog = (serverID: string, dir: string, skip: number): Promise<Commit[]> =>
  App.GitLog(serverID, dir, skip) as Promise<Commit[]>
export const gitDiff = (serverID: string, req: DiffRequest): Promise<Diff> =>
  App.GitDiff(serverID, req)
export const gitBranches = (serverID: string, dir: string): Promise<Branch[]> =>
  App.GitBranches(serverID, dir)
export const gitPlan = (serverID: string, dir: string, op: GitOp, branch: string): Promise<Plan> =>
  App.GitPlan(serverID, dir, op, branch)
/** Runs a confirmed operation; command must be the confirmed plan's. */
export const gitRun = (
  serverID: string,
  dir: string,
  op: GitOp,
  branch: string,
  command: string,
): Promise<OpResult> => App.GitRun(serverID, dir, op, branch, command)
