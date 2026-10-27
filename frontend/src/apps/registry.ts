import { t } from '../lib/i18n/vi'
import EditorApp from './editor/EditorApp.svelte'
import FilesApp from './files/FilesApp.svelte'
import MonitorApp from './monitor/MonitorApp.svelte'
import PlaceholderApp from './PlaceholderApp.svelte'
import SettingsApp from './settings/SettingsApp.svelte'
import TerminalApp from './terminal/TerminalApp.svelte'
import TransfersApp from './transfers/TransfersApp.svelte'
import { EDITOR_SIZE, TERMINAL_SIZE, TRANSFERS_SIZE } from '../lib/wm/intents.svelte'
import type { AppDef, AppId } from './types'

export const apps: AppDef[] = [
  {
    id: 'terminal',
    label: t.apps.terminal,
    icon: 'terminal',
    theme: 'dark',
    size: TERMINAL_SIZE,
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
    size: { w: 860, h: 580 },
    component: MonitorApp,
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
    size: EDITOR_SIZE,
    component: EditorApp,
  },
  {
    id: 'settings',
    label: t.apps.settings,
    icon: 'settings',
    theme: 'light',
    size: { w: 520, h: 380 },
    component: SettingsApp,
  },
  {
    id: 'transfers',
    label: t.apps.transfers,
    icon: 'download',
    theme: 'light',
    size: TRANSFERS_SIZE,
    component: TransfersApp,
    dock: false,
  },
]

/** Apps with a tile in the dock. */
export const dockApps = apps.filter((a) => a.dock !== false)

export function getApp(id: string): AppDef | undefined {
  return apps.find((a) => a.id === id)
}

export type { AppId }
