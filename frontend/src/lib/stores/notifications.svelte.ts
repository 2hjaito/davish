import { onNotifyNew, type NotifyNew } from '../api/events'
import type { LogSource } from '../api/logs'
import { hasRuntime } from '../api/runtime'
import { formatPct } from '../format'
import { t } from '../i18n/vi'

export type Level = 'info' | 'warn' | 'error'

export interface Notification {
  id: string
  level: Level
  title: string
  body: string
  serverID: string
  /** ms since epoch */
  at: number
  /** One button on the banner and in the list, such as "Show in Finder". */
  action?: { label: string; run: () => void }
}

/** The icon beside a notification. */
export function levelIcon(level: Level): 'info' | 'warning' {
  return level === 'info' ? 'info' : 'warning'
}

/** How long a banner stays; errors longer, since they need reading. */
export const BANNER_MS = { info: 6000, warn: 8000, error: 10000 } as const
/** Banners on screen at once; older ones step aside. */
export const MAX_BANNERS = 3

// Mock data matching the mockup, for the browser preview only; the real app
// starts empty and fills from notify:new.
const demo: Notification[] = hasRuntime
  ? []
  : [
      {
        id: 'n1',
        level: 'warn',
        title: 'lightrag trên dangth-vps đang khởi động lại liên tục',
        body: 'Container đã khởi động lại 6 lần.',
        serverID: 'dangth-vps',
        at: Date.now(),
      },
    ]

/** Opens the logs of what a health alert is about. */
export type ShowLogs = (serverID: string, source: LogSource) => void

/**
 * Writes the text of a notify:new payload; serverName labels the server.
 * With showLogs, health alerts get a button to the failing unit's or
 * container's logs.
 */
export function describe(
  p: NotifyNew,
  serverName: string,
  at = Date.now(),
  showLogs?: ShowLogs,
): Notification {
  const base = { id: p.id, level: p.level, serverID: p.serverID, at }
  const target = p.target ?? ''
  const logs = (source: LogSource) =>
    showLogs && { label: t.notifications.showLogs, run: () => showLogs(p.serverID, source) }
  switch (p.kind) {
    case 'service-failed':
      return {
        ...base,
        title: t.notifications.serviceFailed(target.replace(/\.service$/, ''), serverName),
        body: t.notifications.serviceFailedBody,
        action: logs({ kind: 'journal', target }),
      }
    case 'cert-expiring':
      return {
        ...base,
        title:
          p.value < 0
            ? t.notifications.certExpired(serverName)
            : t.notifications.certExpiring(serverName),
        body: t.notifications.certBody(target, p.value),
      }
    case 'container-restarting':
      return {
        ...base,
        title: t.notifications.containerRestarting(target, serverName),
        body: t.notifications.containerRestartingBody(p.value),
        action: logs({ kind: 'docker', target }),
      }
  }
  const title =
    p.kind === 'disk-high'
      ? t.notifications.diskHigh(serverName, p.mount ?? '/')
      : t.notifications.memHigh(serverName)
  return {
    ...base,
    title,
    body: t.notifications.overThreshold(formatPct(p.value), formatPct(p.threshold)),
  }
}

class Notifications {
  /** Newest first. */
  list = $state<Notification[]>(demo)

  unread = $derived(this.list.length)
  /** Ids shown as banners in the corner, newest first. */
  bannerIds = $state<string[]>([])
  banners = $derived(this.bannerIds.flatMap((id) => this.list.find((n) => n.id === id) ?? []))

  // Plain map: timer handles are bookkeeping, not UI state.
  #timers = new Map<string, ReturnType<typeof setTimeout>>()

  /**
   * Listens to notify:new; nameOf turns a server id into the name to show.
   * Returns the function that stops listening.
   */
  init(nameOf: (serverID: string) => string, showLogs?: ShowLogs): () => void {
    if (!hasRuntime) return () => {}
    return onNotifyNew((p) => this.add(describe(p, nameOf(p.serverID), Date.now(), showLogs)))
  }

  /**
   * Adds n on top, replacing an earlier notification with the same id, and
   * shows it as a banner for a few seconds.
   */
  add(n: Notification) {
    this.list = [n, ...this.list.filter((x) => x.id !== n.id)]
    this.bannerIds = [n.id, ...this.bannerIds.filter((id) => id !== n.id)].slice(0, MAX_BANNERS)
    this.#schedule(n.id, BANNER_MS[n.level])
  }

  dismiss(id: string) {
    this.list = this.list.filter((n) => n.id !== id)
    this.hideBanner(id)
  }

  /** Takes the banner away; the notification stays in the list. */
  hideBanner(id: string) {
    clearTimeout(this.#timers.get(id))
    this.#timers.delete(id)
    this.bannerIds = this.bannerIds.filter((x) => x !== id)
  }

  /** While the pointer rests on a banner it stays; leaving gives it a moment more. */
  holdBanner(id: string) {
    clearTimeout(this.#timers.get(id))
  }

  releaseBanner(id: string) {
    this.#schedule(id, 3000)
  }

  #schedule(id: string, ms: number) {
    clearTimeout(this.#timers.get(id))
    this.#timers.set(
      id,
      setTimeout(() => this.hideBanner(id), ms),
    )
  }
}

export const notifications = new Notifications()
