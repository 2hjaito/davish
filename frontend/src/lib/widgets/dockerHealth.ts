// What the Docker health widget makes of `docker ps`, apart from reading it
// (ADR 0026). Pure: the health of a container is already in the Status line
// docker prints, so nothing more is asked of the server.
import type { Container } from '../api/docker'

export type Health = 'healthy' | 'unhealthy' | 'starting' | 'none'

/**
 * The health docker reports in the Status of a container that has a
 * healthcheck: "Up 3 hours (healthy)", "Up 2 minutes (unhealthy)" and
 * "Up 5 seconds (health: starting)". Anything else has none.
 */
export function healthOf(c: Pick<Container, 'status'>): Health {
  const m = /\((healthy|unhealthy|health: starting)\)/.exec(c.status)
  if (!m) return 'none'
  return m[1] === 'health: starting' ? 'starting' : (m[1] as 'healthy' | 'unhealthy')
}

export interface DockerSummary {
  total: number
  running: number
  /** Names of the containers whose healthcheck fails, up to LIST_MAX. */
  unhealthy: string[]
  /** Names of the containers stuck restarting, up to LIST_MAX. */
  restarting: string[]
  /** Containers still within their first healthcheck. */
  starting: number
  /** How many are over LIST_MAX in unhealthy and restarting together. */
  more: number
}

export const LIST_MAX = 20

/** Every container, counted; docker's own state tells running and restarting apart. */
export function summarize(list: readonly Container[]): DockerSummary {
  const unhealthy: string[] = []
  const restarting: string[] = []
  let running = 0
  let starting = 0
  let more = 0
  for (const c of list) {
    if (c.state === 'running') running++
    const bad =
      c.state === 'restarting' ? restarting : healthOf(c) === 'unhealthy' ? unhealthy : null
    if (bad) {
      if (unhealthy.length + restarting.length < LIST_MAX) bad.push(c.name)
      else more++
    }
    if (healthOf(c) === 'starting') starting++
  }
  return { total: list.length, running, unhealthy, restarting, starting, more }
}

/** Is anything wrong: a failing healthcheck or a container that keeps restarting. */
export const hasProblem = (s: DockerSummary) =>
  s.unhealthy.length + s.restarting.length + s.more > 0

/** How many containers have a problem, those past the list included. */
export const problemCount = (s: DockerSummary) => s.unhealthy.length + s.restarting.length + s.more
