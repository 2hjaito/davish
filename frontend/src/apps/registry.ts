import { t } from '../lib/i18n/vi'
import FilesApp from './files/FilesApp.svelte'
import PlaceholderApp from './PlaceholderApp.svelte'
import SettingsApp from './settings/SettingsApp.svelte'
import TerminalApp from './terminal/TerminalApp.svelte'
import type { AppDef, AppId } from './types'

export const apps: AppDef[] = [
  {
    id: 'terminal',
    label: t.apps.terminal,
    icon: 'terminal',
    color: 'var(--app-terminal)',
    theme: 'dark',
    size: { w: 720, h: 470 },
    component: TerminalApp,
  },
  {
    id: 'files',
    label: t.apps.files,
    icon: 'files',
    color: 'var(--app-files)',
    theme: 'light',
    size: { w: 640, h: 440 },
    component: FilesApp,
  },
  {
    id: 'monitor',
    label: t.apps.monitor,
    icon: 'monitor',
    color: 'var(--app-monitor)',
    theme: 'light',
    size: { w: 640, h: 440 },
    component: PlaceholderApp,
  },
  {
    id: 'services',
    label: t.apps.services,
    icon: 'services',
    color: 'var(--app-services)',
    theme: 'light',
    size: { w: 620, h: 420 },
    component: PlaceholderApp,
  },
  {
    id: 'docker',
    label: t.apps.docker,
    icon: 'docker',
    color: 'var(--app-docker)',
    theme: 'light',
    size: { w: 640, h: 420 },
    component: PlaceholderApp,
  },
  {
    id: 'logs',
    label: t.apps.logs,
    icon: 'logs',
    color: 'var(--app-logs)',
    theme: 'dark',
    size: { w: 720, h: 440 },
    component: PlaceholderApp,
  },
  {
    id: 'editor',
    label: t.apps.editor,
    icon: 'editor',
    color: 'var(--app-editor)',
    theme: 'light',
    size: { w: 720, h: 500 },
    component: PlaceholderApp,
  },
  {
    id: 'settings',
    label: t.apps.settings,
    icon: 'settings',
    color: 'var(--app-settings)',
    theme: 'light',
    size: { w: 520, h: 380 },
    component: SettingsApp,
  },
]

export function getApp(id: string): AppDef | undefined {
  return apps.find((a) => a.id === id)
}

export type { AppId }
