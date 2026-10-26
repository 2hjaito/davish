import { hasRuntime } from '../api/runtime'

export type Level = 'info' | 'warn' | 'error'

export interface Notification {
  id: string
  level: Level
  title: string
  body: string
  serverID: string
  at: Date
}

// Mock data matching the mockup, for the browser preview only; the real app
// starts empty until notify:new is wired up (M4/M7).
const demo: Notification[] = hasRuntime
  ? []
  : [
      {
        id: 'n1',
        level: 'warn',
        title: 'lightrag đang khởi động lại liên tục',
        body: 'Container restart 6 lần trong 2 phút.',
        serverID: 'tsc-vps',
        at: new Date(),
      },
    ]

class Notifications {
  list = $state<Notification[]>(demo)

  unread = $derived(this.list.length)

  dismiss(id: string) {
    this.list = this.list.filter((n) => n.id !== id)
  }
}

export const notifications = new Notifications()
