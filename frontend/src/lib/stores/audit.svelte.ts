import { auditEntries } from '../api/audit'
import { type AuditEntry, onAuditEntry } from '../api/events'
import { hasRuntime } from '../api/runtime'

/** Same as Go's ring (internal/audit.Capacity). */
export const CAPACITY = 1000

/** entries with e added in ID order, without duplicates, keeping the newest CAPACITY. */
export function merge(entries: readonly AuditEntry[], add: readonly AuditEntry[]): AuditEntry[] {
  // eslint-disable-next-line svelte/prefer-svelte-reactivity -- a local scratch map, never state
  const byId = new Map(entries.map((e) => [e.id, e]))
  for (const e of add) byId.set(e.id, e)
  const all = [...byId.values()].sort((a, b) => a.id - b.id)
  return all.length > CAPACITY ? all.slice(all.length - CAPACITY) : all
}

// Browser preview only: a few commands like the ones the apps run.
const demo: AuditEntry[] = hasRuntime
  ? []
  : [
      ['services', 'systemctl list-units --type=service --all --no-pager --output=json', 0, 84],
      ['services', 'journalctl -u nginx.service -n 200 --no-pager -o short-iso', 0, 120],
      ['services', 'systemctl restart nginx.service', 1, 61],
      ['docker', "docker ps -a --no-trunc --format '{{json .}}'", 0, 95],
      ['docker', "docker stats --no-stream --no-trunc --format '{{json .}}'", 0, 2048],
      ['logs', 'docker logs -f --tail 200 worker', -1, 0],
      ['files', "rm -- '/home/dangth/old notes.txt'", 0, 40],
    ].map(([app, command, exit, durMs], i) => ({
      id: i + 1,
      time: Date.now() - (7 - i) * 60_000,
      serverID: 'dangth-vps',
      app: app as string,
      command: command as string,
      exit: exit as number,
      durMs: durMs as number,
    }))

class AuditStore {
  entries = $state.raw<AuditEntry[]>(demo)

  /** Loads what Go has and follows audit:entry; returns the unsubscribe. */
  init(): () => void {
    if (!hasRuntime) return () => {}
    const off = onAuditEntry((e) => (this.entries = merge(this.entries, [e])))
    void auditEntries().then((all) => (this.entries = merge(this.entries, all)))
    return off
  }
}

export const audit = new AuditStore()
