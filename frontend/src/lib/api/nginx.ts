// Typed wrappers for the nginx bindings (internal/app/nginx.go).
import * as App from '../../../wailsjs/go/app/App'

export interface NginxServer {
  names: string[]
  listen: string[]
  root?: string
  ssl: boolean
  certs: string[]
  proxyPass: string[]
  /** Where the block starts in its file. */
  line: number
}

export interface NginxCert {
  path: string
  /** Unix seconds; 0 when unreadable. */
  notAfter: number
  names: string[]
  error?: 'unreadable' | 'invalid'
}

export interface NginxSite {
  name: string
  /** The file to open in the Editor. */
  path: string
  /** site: sites-available turned on by a link; conf: conf.d, always on. */
  kind: 'site' | 'conf'
  enabled: boolean
  /** False for conf.d files and files put in sites-enabled directly. */
  toggleable: boolean
  servers: NginxServer[]
  upstreams: string[]
  certs: NginxCert[]
  error?: string
}

export interface TogglePlan {
  name: string
  enable: boolean
  /** Exactly what nginxToggle runs. */
  command: string
  sudo: string
}

export interface NginxIssue {
  level: 'emerg' | 'alert' | 'crit' | 'error' | 'warn'
  message: string
  file?: string
  line?: number
}

export interface NginxTest {
  ok: boolean
  /** Syntax fine as far as nginx got; it stopped on root-only files. */
  needsRoot: boolean
  issues: NginxIssue[]
  output: string
}

export interface NginxReload {
  test: NginxTest
  reloaded: boolean
  needsRoot: boolean
  sudo: string
}

export const nginxList = (serverID: string): Promise<NginxSite[]> =>
  App.NginxList(serverID) as Promise<NginxSite[]>

export const nginxPlanToggle = (
  serverID: string,
  name: string,
  enable: boolean,
): Promise<TogglePlan> => App.NginxPlanToggle(serverID, name, enable)

export const nginxToggle = (serverID: string, plan: TogglePlan): Promise<void> =>
  App.NginxToggle(serverID, plan.name, plan.enable, plan.command)

export const nginxTest = (serverID: string): Promise<NginxTest> =>
  App.NginxTest(serverID) as Promise<NginxTest>

export const nginxReloadCommand = (): Promise<string> => App.NginxReloadCommand()

export const nginxReload = (serverID: string): Promise<NginxReload> =>
  App.NginxReload(serverID) as Promise<NginxReload>

export const nginxCheckCerts = (serverID: string): Promise<void> => App.NginxCheckCerts(serverID)
