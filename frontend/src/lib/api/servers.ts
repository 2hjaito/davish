// Typed wrappers for the server and connection bindings (internal/app/servers.go).
import * as App from '../../../wailsjs/go/app/App'
import { app } from '../../../wailsjs/go/models'

export type AuthMethod = 'agent' | 'key' | 'password'
export type ConnState = 'disconnected' | 'connecting' | 'connected' | 'reconnecting' | 'failed'

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

export const deleteServer = (id: string): Promise<void> => App.DeleteServer(id)
export const reorderServers = (ids: string[]): Promise<void> => App.ReorderServers(ids)
export const connect = (id: string): Promise<void> => App.Connect(id)
export const disconnect = (id: string): Promise<void> => App.Disconnect(id)
export const acceptHostKey = (id: string, fingerprint: string): Promise<void> =>
  App.AcceptHostKey(id, fingerprint)
