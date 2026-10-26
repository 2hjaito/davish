import type { Component } from 'svelte'

import type { IconName } from '../lib/ui/icons'
import type { WindowState } from '../lib/wm/windowManager.svelte'

export type AppId =
  'terminal' | 'files' | 'monitor' | 'services' | 'docker' | 'logs' | 'editor' | 'settings'

export interface AppProps {
  win: WindowState
}

export interface AppDef {
  id: AppId
  label: string
  icon: IconName
  theme: 'light' | 'dark'
  size: { w: number; h: number }
  component: Component<AppProps>
}
