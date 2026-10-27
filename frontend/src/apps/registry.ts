import { t } from '../lib/i18n/vi'
import AuditApp from './audit/AuditApp.svelte'
import CronApp from './cron/CronApp.svelte'
import DockerApp from './docker/DockerApp.svelte'
import EditorApp from './editor/EditorApp.svelte'
import FilesApp from './files/FilesApp.svelte'
import LogsApp from './logs/LogsApp.svelte'
import MonitorApp from './monitor/MonitorApp.svelte'
import ServicesApp from './services/ServicesApp.svelte'
import SettingsApp from './settings/SettingsApp.svelte'
import TerminalApp from './terminal/TerminalApp.svelte'
import TransfersApp from './transfers/TransfersApp.svelte'
import {
  AUDIT_SIZE,
  EDITOR_SIZE,
  LOGS_SIZE,
  TERMINAL_SIZE,
  TRANSFERS_SIZE,
} from '../lib/wm/intents.svelte'
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
    size: { w: 900, h: 580 },
    component: ServicesApp,
  },
  {
    id: 'docker',
    label: t.apps.docker,
    icon: 'docker',
    theme: 'light',
    size: { w: 900, h: 580 },
    component: DockerApp,
  },
  {
    id: 'logs',
    label: t.apps.logs,
    icon: 'logs',
    theme: 'dark',
    size: LOGS_SIZE,
    component: LogsApp,
  },
  {
    id: 'cron',
    label: t.apps.cron,
    icon: 'clock',
    theme: 'light',
    size: { w: 900, h: 580 },
    component: CronApp,
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
  {
    id: 'audit',
    label: t.apps.audit,
    icon: 'list',
    theme: 'light',
    size: AUDIT_SIZE,
    component: AuditApp,
    dock: false,
  },
]

/** Apps with a tile in the dock. */
export const dockApps = apps.filter((a) => a.dock !== false)

export function getApp(id: string): AppDef | undefined {
  return apps.find((a) => a.id === id)
}

export type { AppId }
