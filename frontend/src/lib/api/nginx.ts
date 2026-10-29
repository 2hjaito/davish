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

/** Deleting a site (internal/nginx.RemovePlan). */
export interface RemovePlan {
  path: string
  name: string
  /** nginx serves it until it reloads. */
  enabled: boolean
  /** Let's Encrypt certificates only this site uses. */
  certs: string[]
  withCerts: boolean
  /** Used by other sites too; always kept. */
  sharedCerts: string[]
  /** Exactly what nginxRemove runs. */
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

export const nginxPlanRemove = (
  serverID: string,
  path: string,
  withCerts: boolean,
): Promise<RemovePlan> => App.NginxPlanRemove(serverID, path, withCerts)

/** Runs the confirmed command; FS_STALE when the site changed since. */
export const nginxRemove = (serverID: string, plan: RemovePlan): Promise<void> =>
  App.NginxRemove(serverID, plan.path, plan.withCerts, plan.command)

export const nginxCheckCerts = (serverID: string): Promise<void> => App.NginxCheckCerts(serverID)

/** The new site form (internal/nginx.NewSite). */
export interface NewSite {
  names: string[]
  type: 'static' | 'proxy' | 'php'
  root: string
  /** host:port or a URL. */
  upstream: string
  websocket: boolean
  phpSocket: string
}

export interface SiteOptions {
  /** No sites-available: new sites go to conf.d. */
  confD: boolean
  phpSockets: string[]
}

export interface CreatePlan {
  name: string
  path: string
  config: string
  /** Exactly what nginxCreate runs. */
  command: string
  sudo: string
}

export interface CreateResult {
  created: boolean
  /** nginx -t's verdict; a failing test leaves the site off. */
  test?: NginxTest
  needsRoot: boolean
  sudo: string
}

export const nginxSiteOptions = (serverID: string): Promise<SiteOptions> =>
  App.NginxSiteOptions(serverID)

export const nginxRenderSite = (site: NewSite): Promise<string> => App.NginxRenderSite(site)

export const nginxPlanCreate = (
  serverID: string,
  site: NewSite,
  config: string,
): Promise<CreatePlan> => App.NginxPlanCreate(serverID, site, config)

/** Runs the confirmed command; FS_STALE when anything changed since. */
export const nginxCreate = (
  serverID: string,
  site: NewSite,
  plan: CreatePlan,
): Promise<CreateResult> =>
  App.NginxCreate(serverID, site, plan.config, plan.command) as Promise<CreateResult>

/** Where one domain points, seen from this machine. */
export interface DNSCheck {
  name: string
  ips: string[]
  /** One of ips is the server's. */
  ok: boolean
  error?: string
}

/** Turning on HTTPS with certbot (internal/nginx.SSLPlan). */
export interface SSLPlan {
  names: string[]
  /** Wildcards, IPs, "_": no certificate through certbot --nginx. */
  skipped: string[]
  serverIPs: string[]
  dns: DNSCheck[]
  redirect: boolean
  /** The certbot command for a terminal; empty without names. */
  sudo: string
}

export const nginxPlanSSL = (
  serverID: string,
  names: string[],
  redirect: boolean,
): Promise<SSLPlan> => App.NginxPlanSSL(serverID, names, redirect)
