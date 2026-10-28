import { t } from '../lib/i18n/vi'
import AuditApp from './audit/AuditApp.svelte'
import CronApp from './cron/CronApp.svelte'
import KeysApp from './keys/KeysApp.svelte'
import PackagesApp from './packages/PackagesApp.svelte'
import PortsApp from './ports/PortsApp.svelte'
import DiskApp from './disk/DiskApp.svelte'
import DockerApp from './docker/DockerApp.svelte'
import EditorApp from './editor/EditorApp.svelte'
import FilesApp from './files/FilesApp.svelte'
import GitApp from './git/GitApp.svelte'
import LogsApp from './logs/LogsApp.svelte'
import NginxApp from './nginx/NginxApp.svelte'
import MonitorApp from './monitor/MonitorApp.svelte'
import ServicesApp from './services/ServicesApp.svelte'
import SettingsApp from './settings/SettingsApp.svelte'
import TerminalApp from './terminal/TerminalApp.svelte'
import TransfersApp from './transfers/TransfersApp.svelte'
import {
  AUDIT_SIZE,
  DISK_SIZE,
  EDITOR_SIZE,
  FILES_SIZE,
  LOGS_SIZE,
  SERVICES_SIZE,
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
    size: FILES_SIZE,
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
    size: SERVICES_SIZE,
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
    id: 'nginx',
    label: t.apps.nginx,
    icon: 'brandNginx',
    theme: 'light',
    size: { w: 920, h: 600 },
    component: NginxApp,
  },
  {
    id: 'keys',
    label: t.apps.keys,
    icon: 'key',
    theme: 'light',
    size: { w: 900, h: 580 },
    component: KeysApp,
  },
  {
    id: 'packages',
    label: t.apps.packages,
    icon: 'package',
    theme: 'light',
    size: { w: 960, h: 600 },
    component: PackagesApp,
  },
  {
    id: 'git',
    label: t.apps.git,
    icon: 'gitBranch',
    theme: 'light',
    size: { w: 1100, h: 640 },
    component: GitApp,
  },
  {
    id: 'ports',
    label: t.apps.ports,
    icon: 'network',
    theme: 'light',
    size: { w: 900, h: 580 },
    component: PortsApp,
  },
  {
    id: 'disk',
    label: t.apps.disk,
    icon: 'disk',
    theme: 'light',
    size: DISK_SIZE,
    component: DiskApp,
  },
  {
    id: 'editor',
    label: t.apps.editor,
    icon: 'editor',
    theme: 'light',
    size: EDITOR_SIZE,
    component: EditorApp,
    layout: false,
  },
  {
    id: 'settings',
    label: t.apps.settings,
    icon: 'settings',
    theme: 'light',
    size: { w: 580, h: 600 },
    component: SettingsApp,
    layout: false,
  },
  {
    id: 'transfers',
    label: t.apps.transfers,
    icon: 'download',
    theme: 'light',
    size: TRANSFERS_SIZE,
    component: TransfersApp,
    dock: false,
    layout: false,
  },
  {
    id: 'audit',
    label: t.apps.audit,
    icon: 'list',
    theme: 'light',
    size: AUDIT_SIZE,
    component: AuditApp,
    dock: false,
    layout: false,
  },
]

/** Apps with a tile in the dock. */
export const dockApps = apps.filter((a) => a.dock !== false)

export function getApp(id: string): AppDef | undefined {
  return apps.find((a) => a.id === id)
}

export type { AppId }
