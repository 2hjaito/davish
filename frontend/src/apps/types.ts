import type { Component } from 'svelte'

import type { IconName } from '../lib/ui/icons'
import type { WindowState } from '../lib/wm/windowManager.svelte'

export type AppId =
  | 'terminal'
  | 'files'
  | 'monitor'
  | 'services'
  | 'docker'
  | 'logs'
  | 'cron'
  | 'nginx'
  | 'keys'
  | 'packages'
  | 'git'
  | 'ports'
  | 'disk'
  | 'security'
  | 'browser'
  | 'editor'
  | 'settings'
  | 'transfers'
  | 'audit'

export interface AppProps {
  win: WindowState
}

export interface AppDef {
  id: AppId
  label: string
  icon: IconName
  theme: 'light' | 'dark'
  size: { w: number; h: number }
  /** The app's code, in its own chunk; see loadApp. */
  load: () => Promise<{ default: Component<AppProps> }>
  /** False for helper windows opened from another app, such as Transfers. */
  dock?: boolean
  /**
   * False for windows a server's layout does not reopen: they need context
   * (the Editor's files) or do not belong to one server (Settings).
   */
  layout?: boolean
}
