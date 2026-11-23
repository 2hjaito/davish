import { dockerList } from '../api/docker'
import type { Container } from '../api/docker'
import { hasRuntime } from '../api/runtime'
import { type DockerSummary, summarize } from '../widgets/dockerHealth'
import { pageVisible } from '../wm/pageVisible.svelte'
import { type Deps, PolledReading } from './polledReading.svelte'
import { servers } from './servers.svelte'

export { MIN_GAP_MS, type Problem, type Reading, REFRESH_MS } from './polledReading.svelte'

export interface DockerDeps {
  list: (serverID: string) => Promise<Container[]>
  connected: Deps<never>['connected']
  active: Deps<never>['active']
  now: Deps<never>['now']
}

/** The reading the Docker health widgets share: one `docker ps -a` per watched server. */
export class DockerHealth extends PolledReading<DockerSummary> {
  constructor({ list, ...rest }: DockerDeps) {
    super({ ...rest, read: async (serverID) => summarize(await list(serverID)) })
  }
}

/** Sample containers for the browser preview, which has no server. */
async function demoList(): Promise<Container[]> {
  const c = (name: string, state: string, status: string): Container => ({
    id: name,
    name,
    image: 'demo',
    state,
    status,
    ports: '',
    command: '',
    created: '',
    runningFor: '',
  })
  return [
    c('web', 'running', 'Up 3 days (healthy)'),
    c('api', 'running', 'Up 3 days (healthy)'),
    c('db', 'running', 'Up 3 days (healthy)'),
    c('cache', 'running', 'Up 2 hours (unhealthy)'),
    c('worker', 'restarting', 'Restarting (1) 12 seconds ago'),
    c('backup', 'exited', 'Exited (0) 6 hours ago'),
  ]
}

export const dockerHealth = new DockerHealth({
  list: hasRuntime ? dockerList : demoList,
  connected: (id) => !hasRuntime || servers.list.find((s) => s.id === id)?.state === 'connected',
  active: () => pageVisible.visible && typeof document !== 'undefined' && document.hasFocus(),
  now: Date.now,
})
