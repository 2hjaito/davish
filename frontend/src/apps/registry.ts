import type { Component } from 'svelte'

import { t } from '../lib/i18n'
import {
  AUDIT_SIZE,
  BROWSER_SIZE,
  DISK_SIZE,
  EDITOR_SIZE,
  FILES_SIZE,
  LOGS_SIZE,
  SERVICES_SIZE,
  TERMINAL_SIZE,
  TRANSFERS_SIZE,
} from '../lib/wm/intents.svelte'
import type { AppDef, AppId, AppProps } from './types'

export const apps: AppDef[] = [
  {
    id: 'terminal',
    label: t.apps.terminal,
    icon: 'terminal',
    theme: 'dark',
    size: TERMINAL_SIZE,
    load: () => import('./terminal/TerminalApp.svelte'),
  },
  {
    id: 'files',
    label: t.apps.files,
    icon: 'files',
    theme: 'light',
    size: FILES_SIZE,
    load: () => import('./files/FilesApp.svelte'),
  },
  {
    id: 'monitor',
    label: t.apps.monitor,
    icon: 'monitor',
    theme: 'light',
    size: { w: 860, h: 580 },
    load: () => import('./monitor/MonitorApp.svelte'),
  },
  {
    id: 'services',
    label: t.apps.services,
    icon: 'services',
    theme: 'light',
    size: SERVICES_SIZE,
    load: () => import('./services/ServicesApp.svelte'),
  },
  {
    id: 'docker',
    label: t.apps.docker,
    icon: 'docker',
    theme: 'light',
    size: { w: 900, h: 580 },
    load: () => import('./docker/DockerApp.svelte'),
  },
  {
    id: 'logs',
    label: t.apps.logs,
    icon: 'logs',
    theme: 'dark',
    size: LOGS_SIZE,
    load: () => import('./logs/LogsApp.svelte'),
  },
  {
    id: 'cron',
    label: t.apps.cron,
    icon: 'clock',
    theme: 'light',
    size: { w: 900, h: 580 },
    load: () => import('./cron/CronApp.svelte'),
  },
  {
    id: 'nginx',
    label: t.apps.nginx,
    icon: 'brandNginx',
    theme: 'light',
    size: { w: 920, h: 600 },
    load: () => import('./nginx/NginxApp.svelte'),
  },
  {
    id: 'keys',
    label: t.apps.keys,
    icon: 'key',
    theme: 'light',
    size: { w: 900, h: 580 },
    load: () => import('./keys/KeysApp.svelte'),
  },
  {
    id: 'packages',
    label: t.apps.packages,
    icon: 'package',
    theme: 'light',
    size: { w: 960, h: 600 },
    load: () => import('./packages/PackagesApp.svelte'),
  },
  {
    id: 'git',
    label: t.apps.git,
    icon: 'gitBranch',
    theme: 'light',
    size: { w: 1100, h: 640 },
    load: () => import('./git/GitApp.svelte'),
  },
  {
    id: 'ports',
    label: t.apps.ports,
    icon: 'network',
    theme: 'light',
    size: { w: 900, h: 580 },
    load: () => import('./ports/PortsApp.svelte'),
  },
  {
    id: 'browser',
    label: t.apps.browser,
    icon: 'globe2',
    theme: 'light',
    size: BROWSER_SIZE,
    load: () => import('./browser/BrowserApp.svelte'),
    layout: false,
  },
  {
    id: 'disk',
    label: t.apps.disk,
    icon: 'disk',
    theme: 'light',
    size: DISK_SIZE,
    load: () => import('./disk/DiskApp.svelte'),
  },
  {
    id: 'security',
    label: t.apps.security,
    icon: 'shield',
    theme: 'light',
    size: { w: 820, h: 620 },
    load: () => import('./security/SecurityApp.svelte'),
  },
  {
    id: 'editor',
    label: t.apps.editor,
    icon: 'editor',
    theme: 'light',
    size: EDITOR_SIZE,
    load: () => import('./editor/EditorApp.svelte'),
    layout: false,
  },
  {
    id: 'settings',
    label: t.apps.settings,
    icon: 'settings',
    theme: 'light',
    size: { w: 580, h: 600 },
    load: () => import('./settings/SettingsApp.svelte'),
    layout: false,
  },
  {
    id: 'transfers',
    label: t.apps.transfers,
    icon: 'download',
    theme: 'light',
    size: TRANSFERS_SIZE,
    load: () => import('./transfers/TransfersApp.svelte'),
    dock: false,
    layout: false,
  },
  {
    id: 'audit',
    label: t.apps.audit,
    icon: 'list',
    theme: 'light',
    size: AUDIT_SIZE,
    load: () => import('./audit/AuditApp.svelte'),
    dock: false,
    layout: false,
  },
  {
    id: 'database',
    label: t.apps.database,
    icon: 'disk',
    theme: 'light',
    size: { w: 640, h: 560 },
    load: () => import('./database/DatabaseApp.svelte'),
    dock: false,
  },
  {
    id: 'deploy',
    label: t.apps.deploy,
    icon: 'play',
    theme: 'light',
    size: { w: 720, h: 620 },
    load: () => import('./deploy/DeployApp.svelte'),
    dock: false,
    layout: false,
  },
]

/** Apps with a tile in the dock. */
export const dockApps = apps.filter((a) => a.dock !== false)

export function getApp(id: string): AppDef | undefined {
  return apps.find((a) => a.id === id)
}

// Each app's code loads the first time one of its windows opens, so the
// first screen carries only the shell (docs/PLAN.md M21).
const loaded = new Map<AppId, Promise<Component<AppProps>>>()

/** The app's component, loading its chunk once. */
export function loadApp(app: AppDef): Promise<Component<AppProps>> {
  let p = loaded.get(app.id)
  if (!p) {
    p = app.load().then((m) => m.default)
    // A failed load (an update replaced the chunk) may be retried.
    p.catch(() => loaded.delete(app.id))
    loaded.set(app.id, p)
  }
  return p
}

export type { AppId }
