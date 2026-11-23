// Typed wrappers for the server and connection bindings (internal/app/servers.go).
import * as App from '../../../wailsjs/go/app/App'
import { app } from '../../../wailsjs/go/models'

export type AuthMethod = 'agent' | 'key' | 'password'
export type ConnState = 'disconnected' | 'connecting' | 'connected' | 'reconnecting' | 'failed'

/** The operating system of a server; 'unknown' until it has answered. */
export type Platform = 'linux' | 'windows' | 'darwin' | 'unknown'

/** A saved server as Go reports it: never a secret, only whether one is stored. */
export interface ServerInfo {
  id: string
  name: string
  host: string
  port: number
  user: string
  auth: AuthMethod
  keyPath?: string
  color?: string
  jumpID?: string
  controlPath?: string
  hasPassword: boolean
  hasPassphrase: boolean
  state: ConnState
}

/** What the add/edit dialog sends; empty password/passphrase keeps the stored one. */
export interface ServerDraft {
  id: string
  name: string
  host: string
  port: number
  user: string
  auth: AuthMethod
  keyPath: string
  color: string
  jumpID: string
  controlPath: string
  password: string
  passphrase: string
}

function info(v: app.ServerView): ServerInfo {
  return { ...v, auth: v.auth as AuthMethod, state: v.state as ConnState }
}

export async function listServers(): Promise<ServerInfo[]> {
  return (await App.ListServers()).map(info)
}

export async function saveServer(draft: ServerDraft): Promise<ServerInfo> {
  return info(await App.SaveServer(app.ServerInput.createFrom(draft)))
}

/** A concrete Host from ~/.ssh/config, as the import dialog lists it. */
export interface ConfigHost {
  alias: string
  hostName: string
  user: string
  port: number
  identityFile?: string
  proxyJump?: string
  exists: boolean
}

export const listConfigHosts = (): Promise<ConfigHost[]> => App.SSHConfigHosts()

export async function importConfigHosts(aliases: string[]): Promise<ServerInfo[]> {
  return (await App.ImportSSHHosts(aliases)).map(info)
}

/** Why an inventory host is skipped, or what of it is left out (internal/ansibleinv). */
export interface InventoryIssue {
  code: 'connection' | 'jinja' | 'vault' | 'port' | 'range' | 'password' | 'sshArgs'
  /** The variable or value concerned; never a secret. */
  detail?: string
}

/** A host of an Ansible inventory, as the import dialog lists it. */
export interface InventoryHost {
  name: string
  groups: string[]
  address: string
  user: string
  port: number
  keyFile?: string
  proxyJump?: string
  notes?: InventoryIssue[]
  exists: boolean
}

export interface InventorySkipped {
  name: string
  groups: string[]
  issue: InventoryIssue
}

export interface InventoryListing {
  path: string
  hosts: InventoryHost[]
  skipped: InventorySkipped[]
  groups: string[]
}

/** The inventory Ansible would use on this machine, '' when none. */
export const defaultInventory = (): Promise<string> => App.AnsibleDefaultInventory()

/** Asks for an inventory file; '' when canceled. */
export const pickInventory = (title: string): Promise<string> => App.AnsiblePickInventory(title)

export const listInventory = async (path: string): Promise<InventoryListing> =>
  (await App.AnsibleInventoryHosts(path)) as InventoryListing

export async function importInventoryHosts(path: string, names: string[]): Promise<ServerInfo[]> {
  return (await App.ImportAnsibleHosts(path, names)).map(info)
}

export const deleteServer = (id: string): Promise<void> => App.DeleteServer(id)
export const reorderServers = (ids: string[]): Promise<void> => App.ReorderServers(ids)
export const connect = (id: string): Promise<void> => App.Connect(id)
export const disconnect = (id: string): Promise<void> => App.Disconnect(id)
export const acceptHostKey = (id: string, fingerprint: string): Promise<void> =>
  App.AcceptHostKey(id, fingerprint)
