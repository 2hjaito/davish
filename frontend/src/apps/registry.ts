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
    theme: 'dark',
    size: { w: 720, h: 470 },
    component: TerminalApp,
  },
  {
    id: 'files',
    label: t.apps.files,
    icon: 'files',
    theme: 'light',
    size: { w: 640, h: 440 },
    component: FilesApp,
  },
  {
    id: 'monitor',
    label: t.apps.monitor,
    icon: 'monitor',
    theme: 'light',
    size: { w: 640, h: 440 },
    component: PlaceholderApp,
  },
  {
    id: 'services',
    label: t.apps.services,
    icon: 'services',
    theme: 'light',
    size: { w: 620, h: 420 },
    component: PlaceholderApp,
  },
  {
    id: 'docker',
    label: t.apps.docker,
    icon: 'docker',
    theme: 'light',
    size: { w: 640, h: 420 },
    component: PlaceholderApp,
  },
  {
    id: 'logs',
    label: t.apps.logs,
    icon: 'logs',
    theme: 'dark',
    size: { w: 720, h: 440 },
    component: PlaceholderApp,
  },
  {
    id: 'editor',
    label: t.apps.editor,
    icon: 'editor',
    theme: 'light',
    size: { w: 720, h: 500 },
    component: PlaceholderApp,
  },
  {
    id: 'settings',
    label: t.apps.settings,
    icon: 'settings',
    theme: 'light',
    size: { w: 520, h: 380 },
    component: SettingsApp,
  },
]

export function getApp(id: string): AppDef | undefined {
  return apps.find((a) => a.id === id)
}

export type { AppId }
