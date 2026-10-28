// Typed wrappers for the SSH key bindings (internal/app/keys.go).
import * as App from '../../../wailsjs/go/app/App'

/** A key allowed to log in to the server (internal/sshkeys.AuthorizedKey). */
export interface AuthorizedKey {
  type: string
  fingerprint: string
  comment: string
  options: string[]
  /** Davish may log in with it: removing it can lock Davish out. */
  login: boolean
  text: string
}

export interface Authorized {
  path: string
  keys: AuthorizedKey[]
}

/** A change to authorized_keys, for the confirmation. */
export interface KeyPlan {
  fingerprint: string
  comment: string
  login: boolean
  backup?: string
  command: string
}

/** A key pair in the server user's ~/.ssh. */
export interface ServerKey {
  name: string
  path: string
  type: string
  fingerprint: string
  comment: string
  public: string
}

export interface CreatePlan {
  name: string
  path: string
  comment: string
  command: string
}

export type GitHost = 'github.com' | 'gitlab.com'
export const GIT_HOSTS: GitHost[] = ['github.com', 'gitlab.com']

export interface GitTestPlan {
  host: GitHost
  key: string
  command: string
}

export type GitOutcome = 'ok' | 'denied' | 'hostkey' | 'unreachable' | 'other'

export interface GitTestResult {
  outcome: GitOutcome
  account?: string
  /** The host's own words, shown as text. */
  output: string
}

/** A key pair in ~/.ssh on this machine. */
export interface LocalKey {
  name: string
  path: string
  type: string
  fingerprint: string
  comment: string
  public: string
  encrypted: boolean
}

export const keysAuthorized = (serverID: string): Promise<Authorized> =>
  App.KeysAuthorized(serverID)
export const keysPlanAdd = (serverID: string, text: string): Promise<KeyPlan> =>
  App.KeysPlanAdd(serverID, text)
export const keysAdd = (serverID: string, text: string, command: string): Promise<void> =>
  App.KeysAdd(serverID, text, command)
export const keysPlanRemove = (serverID: string, fingerprint: string): Promise<KeyPlan> =>
  App.KeysPlanRemove(serverID, fingerprint)
export const keysRemove = (serverID: string, plan: KeyPlan): Promise<void> =>
  App.KeysRemove(serverID, plan)
export const keysServer = (serverID: string): Promise<ServerKey[]> => App.KeysServer(serverID)
export const keysPlanCreate = (
  serverID: string,
  name: string,
  comment: string,
): Promise<CreatePlan> => App.KeysPlanCreate(serverID, name, comment)
export const keysCreate = (
  serverID: string,
  name: string,
  comment: string,
  command: string,
): Promise<void> => App.KeysCreate(serverID, name, comment, command)
export const keysPlanGitTest = (
  serverID: string,
  host: GitHost,
  name: string,
): Promise<GitTestPlan> => App.KeysPlanGitTest(serverID, host, name) as Promise<GitTestPlan>
export const keysGitTest = (
  serverID: string,
  host: GitHost,
  name: string,
  command: string,
): Promise<GitTestResult> =>
  App.KeysGitTest(serverID, host, name, command) as Promise<GitTestResult>
export const keysLocal = (): Promise<LocalKey[]> => App.KeysLocal()
/** Never overwrites; FS_EXISTS when the name is taken. */
export const keysGenerateLocal = (
  name: string,
  comment: string,
  passphrase: string,
): Promise<LocalKey> => App.KeysGenerateLocal(name, comment, passphrase)
