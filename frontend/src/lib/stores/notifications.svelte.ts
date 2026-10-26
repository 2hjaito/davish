export type Level = 'info' | 'warn' | 'error'

export interface Notification {
  id: string
  level: Level
  title: string
  body: string
  serverID: string
  at: Date
}

class Notifications {
  // Mock data matching the mockup until notify:new is wired up.
  list = $state<Notification[]>([
    {
      id: 'n1',
      level: 'warn',
      title: 'lightrag đang khởi động lại liên tục',
      body: 'Container restart 6 lần trong 2 phút.',
      serverID: 'tsc-vps',
      at: new Date(),
    },
  ])

  unread = $derived(this.list.length)

  dismiss(id: string) {
    this.list = this.list.filter((n) => n.id !== id)
  }
}

export const notifications = new Notifications()
