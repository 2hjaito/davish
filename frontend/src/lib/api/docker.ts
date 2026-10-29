// Typed wrappers for the Docker bindings (internal/app/docker.go).
import * as App from '../../../wailsjs/go/app/App'
import { docker } from '../../../wailsjs/go/models'

/** rm deletes a stopped container. */
export type ContainerAction = 'start' | 'stop' | 'restart' | 'rm'

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
  /** Docker's "6 days ago", in English. */
  runningFor: string
  /** docker compose project and service, when started by compose. */
  project?: string
  service?: string
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

/** One row of docker images. */
export interface Image {
  /** sha256:…, full */
  id: string
  repository: string
  tag: string
  created: string
  /** "2 days ago", in English from docker */
  createdSince: string
  size: number
  /** Containers, running or not, created from it. */
  containers: number
  /** Untagged: no tag points at it any more. */
  dangling: boolean
}

export interface Layer {
  createdBy: string
  size: number
  comment?: string
}

export interface ImageDetail {
  /** Oldest first, as the Dockerfile reads. */
  layers: Layer[]
  architecture: string
  os: string
  entrypoint?: string[]
  cmd?: string[]
  workingDir: string
  user: string
  env?: string[]
  exposedPorts?: string[]
}

export interface ImagePlan {
  ref: string
  /** Exactly what dockerRemoveImage runs. */
  command: string
  /** The docker run line typed into a terminal. */
  run: string
}

/** How commands name an image: repo:tag, or its ID when untagged. */
export const imageRef = (i: Image): string => (i.dangling ? i.id : `${i.repository}:${i.tag}`)

export const dockerImages = (serverID: string): Promise<Image[]> => App.DockerImages(serverID)
export const dockerImage = (serverID: string, ref: string): Promise<ImageDetail> =>
  App.DockerImage(serverID, ref)
export const dockerPlanImage = (ref: string): Promise<ImagePlan> => App.DockerPlanImage(ref)
export const dockerRemoveImage = (serverID: string, ref: string, command: string): Promise<void> =>
  App.DockerRemoveImage(serverID, ref, command)

/** One docker volume. */
export interface Volume {
  name: string
  driver: string
  mountpoint: string
  created: string
  /** -1 when docker cannot tell. */
  size: number
  containers: number
  /** Named by docker with a hash, for one container. */
  anonymous: boolean
  project?: string
}

export interface VolumePlan {
  name: string
  command: string
}

export const dockerVolumes = (serverID: string): Promise<Volume[]> => App.DockerVolumes(serverID)
export const dockerPlanVolume = (name: string): Promise<VolumePlan> => App.DockerPlanVolume(name)
export const dockerRemoveVolume = (
  serverID: string,
  name: string,
  command: string,
): Promise<void> => App.DockerRemoveVolume(serverID, name, command)

/** A published port; host empty means 127.0.0.1. */
export interface PortMap {
  host: string
  hostPort: number
  containerPort: number
  udp: boolean
}

/** A volume name or server path mounted into the container. */
export interface Mount {
  source: string
  target: string
  readOnly: boolean
}

/** The new container form (internal/docker.RunSpec). */
export interface RunSpec {
  name: string
  image: string
  ports: PortMap[]
  /** KEY=value lines. */
  env: string[]
  envFile: string
  mounts: Mount[]
  restart: '' | 'no' | 'always' | 'unless-stopped' | 'on-failure'
  network: string
  args: string[]
}

export interface RunPlan {
  /** The docker run line typed into a terminal. */
  command: string
  /** Ports bound to every address. */
  public: string[]
}

export const dockerPullCommand = (ref: string): Promise<string> => App.DockerPullCommand(ref)

export const dockerPlanRun = (spec: RunSpec): Promise<RunPlan> =>
  App.DockerPlanRun(docker.RunSpec.createFrom(spec))
