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

/** The clone form (internal/git.CloneSpec). */
export interface CloneSpec {
  url: string
  /** Empty only reads the URL. */
  dir: string
  branch: string
}

export interface ClonePlan {
  /** The clone line typed into a terminal; empty without a folder. */
  command: string
  ssh: boolean
  host: string
  /** The host's published SSH key is added to known_hosts first. */
  pinned: boolean
  name: string
}

export const gitPlanClone = (spec: CloneSpec): Promise<ClonePlan> => App.GitPlanClone(spec)

export const gitInitCommand = (dir: string, remote: string, url: string): Promise<string> =>
  App.GitInitCommand(dir, remote, url)

export const gitRemoteCommand = (dir: string, remote: string, url: string): Promise<string> =>
  App.GitRemoteCommand(dir, remote, url)

/** A deploy step Davish offers, its command ready to run in the repository. */
export interface DeployOption {
  /** A script name, a pm2 process, a systemd unit, a compose project. */
  name: string
  command: string
  /** The command asks for a password in the terminal. */
  sudo: boolean
}

export type RunnerKind = 'pm2' | 'systemd' | 'compose'

/** Something running the app now, found on the server. */
export interface AppRunner extends DeployOption {
  kind: RunnerKind
}

/** What a deploy of a repository can be made of (internal/git.DeployInfo). */
export interface DeployInfo {
  pull: DeployOption
  install: DeployOption | null
  builds: DeployOption[]
  runners: AppRunner[]
  /** The folder has a compose file. */
  compose: boolean
}

export const gitDeployInfo = (serverID: string, dir: string): Promise<DeployInfo> =>
  App.GitDeployInfo(serverID, dir) as Promise<DeployInfo>

/** A step to run: the command, and the words shown for it. */
export interface DeployStep {
  label: string
  command: string
}

/** A recipe as one terminal line (internal/git.Deploy). */
export interface DeployLine {
  line: string
  /** Tags the progress marks the line prints (docs/adr/0012). */
  nonce: string
  /** Labels of the steps that run. */
  steps: string[]
  /** A step runs sudo: the user presses Enter and types the password. */
  sudo: boolean
}

/** Deploy steps as one terminal line that stops at a failure. */
export const gitDeployLine = (dir: string, steps: DeployStep[]): Promise<DeployLine> =>
  App.GitDeployLine(dir, steps)
