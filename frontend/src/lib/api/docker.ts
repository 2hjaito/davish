// Typed wrappers for the Docker bindings (internal/app/docker.go).
import * as App from '../../../wailsjs/go/app/App'

export type ContainerAction = 'start' | 'stop' | 'restart'

/** One row of docker ps -a. */
export interface Container {
  id: string
  name: string
  image: string
  /** created, running, restarting, paused, exited, removing, dead */
  state: string
  /** Docker's own summary, in English: "Up 3 hours", "Exited (1) 2 minutes ago". */
  status: string
  ports: string
  command: string
  created: string
}

export interface ContainerStats {
  id: string
  /** Percent of one core; can pass 100. */
  cpu: number
  memUsed: number
  memLimit: number
  pids: number
}

export interface ContainerDetail {
  name: string
  image: string
  state: string
  exitCode: number
  /** Docker's message when the container could not start. */
  error: string
  oomKilled: boolean
  /** healthy, unhealthy, starting; empty without a health check. */
  health: string
  /** RFC 3339, empty if it never ran. */
  startedAt: string
  restarts: number
  /** no, always, unless-stopped, on-failure */
  policy: string
  ports: string[]
  mounts: string[]
  /** The whole docker inspect document, indented. */
  json: string
}

export interface LogLine {
  /** RFC 3339 with nanoseconds. */
  time: string
  text: string
  stderr: boolean
}

export interface ActionPlan {
  name: string
  action: ContainerAction
  /** Exactly what dockerAct runs; show it before stop and restart. */
  command: string
  confirm: boolean
  /** The command with sudo, for a terminal when the server needs root. */
  sudo: string
}

export const dockerList = (serverID: string): Promise<Container[]> => App.DockerList(serverID)

export const dockerStats = (serverID: string): Promise<ContainerStats[]> =>
  App.DockerStats(serverID)

export const dockerInspect = (serverID: string, name: string): Promise<ContainerDetail> =>
  App.DockerInspect(serverID, name)

export const dockerLogs = (serverID: string, name: string): Promise<LogLine[]> =>
  App.DockerLogs(serverID, name).then((l) => l.lines)

export const dockerPlan = (name: string, action: ContainerAction): Promise<ActionPlan> =>
  App.DockerPlan(name, action) as Promise<ActionPlan>

export const dockerAct = (serverID: string, name: string, action: ContainerAction): Promise<void> =>
  App.DockerAct(serverID, name, action)
