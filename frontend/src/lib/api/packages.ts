// Typed wrappers for the package bindings (internal/app/packages.go).
import * as App from '../../../wailsjs/go/app/App'

import { hasRuntime } from './runtime'

export type Manager = 'apt' | 'dnf' | 'yum' | 'apk'

export interface Package {
  name: string
  version: string
  summary: string
  /** Bytes; 0 when unknown. */
  size: number
}

export interface PackageUpdate {
  name: string
  current: string
  available: string
  security: boolean
}

export interface Inventory {
  manager: Manager
  installed: Package[]
  updates: PackageUpdate[]
  /** Seconds since apt fetched its lists; -1 when it never did. */
  listsAge: number
}

export type AppKind = 'runtime' | 'server' | 'tool'

/** A program found on the server (internal/packages.App). */
export interface ServerApp {
  id: string
  label: string
  kind: AppKind
  version: string
  path: string
  configs: string[]
  /** systemd units it may run as, most likely first. */
  units: string[]
  /** nvm or pyenv versions, or php-fpm binaries. */
  versions: string[]
}

export interface NpmGlobal {
  name: string
  version: string
}

/** A tool an app offers to install when the server lacks it. */
export type SetupTool = 'docker' | 'nginx' | 'git' | 'mariadb' | 'postgres'

/** How to install a tool (internal/packages.Setup). */
export interface Setup {
  tool: SetupTool
  /** '' when the server has no package manager Davish knows. */
  manager: Manager | ''
  /** '' when the user has to install it themselves. */
  command: string
  /** The command adds the user to a group: reconnect afterwards. */
  relogin: boolean
}

export type PackageOp =
  'install' | 'remove' | 'upgrade' | 'upgrade-all' | 'upgrade-security' | 'refresh'

export const packagesInventory = (serverID: string): Promise<Inventory> =>
  App.PackagesInventory(serverID) as Promise<Inventory>
export const packagesApps = (serverID: string): Promise<ServerApp[]> =>
  App.PackagesApps(serverID) as Promise<ServerApp[]>
export const packagesNpmGlobals = (serverID: string): Promise<NpmGlobal[]> =>
  App.PackagesNpmGlobals(serverID)
/** The sudo command to type into a terminal; nothing runs. */
export const packagesPlan = (manager: Manager, op: PackageOp, names: string[]): Promise<string> =>
  App.PackagesPlan(manager, op, names)
export const packagesCheckSecurity = (serverID: string): Promise<void> =>
  App.PackagesCheckSecurity(serverID)

// The browser preview has no server: an Ubuntu one, as a stand-in.
const mockSetup: Record<SetupTool, string> = {
  docker:
    'sudo apt-get update && sudo apt-get install docker.io && sudo systemctl enable --now docker && sudo usermod -aG docker "$USER"',
  nginx: 'sudo apt-get update && sudo apt-get install nginx && sudo systemctl enable --now nginx',
  git: 'sudo apt-get update && sudo apt-get install git',
  mariadb:
    'sudo apt-get update && sudo apt-get install mariadb-server && sudo systemctl enable --now mariadb',
  postgres:
    'sudo apt-get update && sudo apt-get install postgresql && sudo systemctl enable --now postgresql',
}

/** The command that installs tool, to type into a terminal; nothing runs. */
export const packagesSetup = (serverID: string, tool: SetupTool): Promise<Setup> =>
  hasRuntime
    ? (App.PackagesSetup(serverID, tool) as Promise<Setup>)
    : Promise.resolve({
        tool,
        manager: 'apt',
        command: mockSetup[tool],
        relogin: tool === 'docker',
      })
