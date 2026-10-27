import { onNotifyNew, type NotifyNew } from '../api/events'
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
}

// Mock data matching the mockup, for the browser preview only; the real app
// starts empty and fills from notify:new.
const demo: Notification[] = hasRuntime
  ? []
  : [
      {
        id: 'n1',
        level: 'warn',
        title: 'lightrag đang khởi động lại liên tục',
        body: 'Container restart 6 lần trong 2 phút.',
        serverID: 'tsc-vps',
        at: Date.now(),
      },
    ]

/** Writes the text of a notify:new payload; serverName labels the server. */
export function describe(p: NotifyNew, serverName: string, at = Date.now()): Notification {
  const title =
    p.kind === 'disk-high'
      ? t.notifications.diskHigh(serverName, p.mount ?? '/')
      : t.notifications.memHigh(serverName)
  return {
    id: p.id,
    level: p.level,
    title,
    body: t.notifications.overThreshold(formatPct(p.value), formatPct(p.threshold)),
    serverID: p.serverID,
    at,
  }
}

class Notifications {
  /** Newest first. */
  list = $state<Notification[]>(demo)

  unread = $derived(this.list.length)

  /**
   * Listens to notify:new; nameOf turns a server id into the name to show.
   * Returns the function that stops listening.
   */
  init(nameOf: (serverID: string) => string): () => void {
    if (!hasRuntime) return () => {}
    return onNotifyNew((p) => this.add(describe(p, nameOf(p.serverID))))
  }

  /** Adds n on top, replacing an earlier notification with the same id. */
  add(n: Notification) {
    this.list = [n, ...this.list.filter((x) => x.id !== n.id)]
  }

  dismiss(id: string) {
    this.list = this.list.filter((n) => n.id !== id)
  }
}

export const notifications = new Notifications()
