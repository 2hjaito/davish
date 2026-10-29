// Typed wrappers for the deploy wizard binding (internal/app/deploy.go).
import * as App from '../../../wailsjs/go/app/App'
import { deploy } from '../../../wailsjs/go/models'

/** The systemd service form (internal/deploy.Unit). */
export interface DeployUnit {
  name: string
  execStart: string
  user: string
  workingDir: string
  env: string[]
  envFile: string
  restart: 'always' | 'on-failure' | 'no'
}

/** The wizard's answers (internal/deploy.Spec). */
export interface DeploySpec {
  source: 'git' | 'image' | 'folder'
  repoURL: string
  branch: string
  image: string
  dir: string
  run: 'systemd' | 'compose' | 'static' | 'container'
  port: number
  containerPort: number
  name: string
  unit: DeployUnit
  domains: string[]
  confD: boolean
  ssl: boolean
  redirect: boolean
}

export interface DeployStep {
  title: string
  command: string
  file?: string
}

export interface DeployPlan {
  steps: DeployStep[]
  /** Runs the steps in a terminal, stopping at the first failure. */
  line: string
  created: string[]
  /** Removes what the steps made. */
  uninstall: string
}

export const deployPlan = (spec: DeploySpec): Promise<DeployPlan> =>
  App.DeployPlan(deploy.Spec.createFrom(spec)) as Promise<DeployPlan>
