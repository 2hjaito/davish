import type { Unit } from '../api/services'
import { servicesList } from '../api/services'
import { hasRuntime } from '../api/runtime'
import { type ServicesSummary, summarize } from '../widgets/servicesHealth'
import { pageVisible } from '../wm/pageVisible.svelte'
import { type Deps, PolledReading } from './polledReading.svelte'
import { servers } from './servers.svelte'

export interface ServicesDeps {
  list: (serverID: string) => Promise<Unit[]>
  connected: Deps<never>['connected']
  active: Deps<never>['active']
  now: Deps<never>['now']
}

/** The reading the Services widgets share: one `systemctl list-units` per watched server. */
export class ServicesHealth extends PolledReading<ServicesSummary> {
  constructor({ list, ...rest }: ServicesDeps) {
    super({ ...rest, read: async (serverID) => summarize(await list(serverID)) })
  }
}

/** Sample units for the browser preview, which has no server. */
async function demoList(): Promise<Unit[]> {
  const u = (name: string, active: string, sub: string): Unit => ({
    name: `${name}.service`,
    load: 'loaded',
    active,
    sub,
    description: '',
  })
  return [
    ...['nginx', 'cron', 'ssh', 'docker', 'redis-server', 'postgresql', 'rsyslog', 'dbus'].map(
      (n) => u(n, 'active', 'running'),
    ),
    ...['cloud-init', 'ufw', 'apparmor'].map((n) => u(n, 'active', 'exited')),
    u('backup', 'failed', 'failed'),
    u('worker', 'activating', 'auto-restart'),
    ...['getty@tty2', 'plymouth', 'snapd', 'fwupd', 'motd-news', 'e2scrub', 'lvm2-monitor'].map(
      (n) => u(n, 'inactive', 'dead'),
    ),
  ]
}

export const servicesHealth = new ServicesHealth({
  list: hasRuntime ? servicesList : demoList,
  connected: (id) => !hasRuntime || servers.list.find((s) => s.id === id)?.state === 'connected',
  active: () => pageVisible.visible && typeof document !== 'undefined' && document.hasFocus(),
  now: Date.now,
})
