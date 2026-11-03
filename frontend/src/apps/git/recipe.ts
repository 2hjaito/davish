// Deploy recipes live on this machine (settings.json), per server and
// repository: never written to the server. They share one setting, keyed
// by server and folder, as setting names cannot hold a path. A recipe
// keeps the user's choices, not commands: the commands come from what
// the server shows each time (internal/git.DeployInfo).
import type { AppRunner, DeployInfo, DeployStep, RunnerKind } from '../../lib/api/git'
import { load, save } from '../../lib/stores/persist'

const SETTING = 'deploy-plans'
/** Recipes before plans: lists of commands, kept as custom steps. */
const LEGACY = 'deploy-recipes'
const key = (serverID: string, dir: string) => `${serverID}:${dir}`

export interface DeployPlan {
  pull: boolean
  install: boolean
  /** The build option's name; empty for no build. */
  build: string
  /** runnerKey of what restarts the app; empty for none. */
  restart: string
  /** The user's own commands, run after the build and before the restart. */
  custom: string[]
}

export const runnerKey = (r: AppRunner) => `${r.kind}:${r.name}`

/** A first plan: everything the repository and server point to. */
export function suggestPlan(info: DeployInfo): DeployPlan {
  const runner = info.runners[0]
  // Compose installs and builds inside its images.
  const compose = runner?.kind === 'compose'
  return {
    pull: true,
    install: info.install !== null && !compose,
    build: !compose && info.builds.some((b) => b.name === 'build') ? 'build' : '',
    restart: runner ? runnerKey(runner) : '',
    custom: [],
  }
}

export interface Loaded {
  plan: DeployPlan
  saved: boolean
  /** The saved restart no longer runs the app, so it was dropped. */
  lostRestart: string
}

/** The saved plan checked against info, or a suggested one. */
export function loadPlan(serverID: string, dir: string, info: DeployInfo): Loaded {
  const k = key(serverID, dir)
  let plan = load<Record<string, DeployPlan>>(SETTING, {})[k]
  if (!plan) {
    const legacy = load<Record<string, string[]>>(LEGACY, {})[k]
    const custom = (legacy ?? []).map((s) => s.trim()).filter((s) => s && !s.startsWith('#'))
    if (custom.length > 0) plan = { pull: false, install: false, build: '', restart: '', custom }
  }
  if (!plan) return { plan: suggestPlan(info), saved: false, lostRestart: '' }
  plan = { ...plan, custom: [...plan.custom] }
  let lostRestart = ''
  if (plan.restart && !info.runners.some((r) => runnerKey(r) === plan.restart)) {
    lostRestart = plan.restart.slice(plan.restart.indexOf(':') + 1)
    plan.restart = ''
  }
  if (plan.build && !info.builds.some((b) => b.name === plan.build)) plan.build = ''
  if (!info.install) plan.install = false
  return { plan, saved: true, lostRestart }
}

export function savePlan(serverID: string, dir: string, plan: DeployPlan): void {
  save(SETTING, { ...load<Record<string, DeployPlan>>(SETTING, {}), [key(serverID, dir)]: plan })
}

/** The words shown for each kind of step. */
export interface StepLabels {
  pull: string
  install: string
  build: (name: string) => string
  restart: (kind: RunnerKind, name: string) => string
}

/** The steps a plan runs, in order. */
export function stepsOf(plan: DeployPlan, info: DeployInfo, l: StepLabels): DeployStep[] {
  const steps: DeployStep[] = []
  if (plan.pull) steps.push({ label: l.pull, command: info.pull.command })
  if (plan.install && info.install) steps.push({ label: l.install, command: info.install.command })
  const build = info.builds.find((b) => b.name === plan.build)
  if (build) steps.push({ label: l.build(build.name), command: build.command })
  for (const c of plan.custom) if (c.trim()) steps.push({ label: '', command: c.trim() })
  const runner = info.runners.find((r) => runnerKey(r) === plan.restart)
  if (runner) steps.push({ label: l.restart(runner.kind, runner.name), command: runner.command })
  return steps
}
