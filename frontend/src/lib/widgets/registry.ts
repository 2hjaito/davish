// The widgets Davish can show (ADR 0026). A kind is listed here and in
// kinds.ts; its component loads on first use, as an app's does, so none of
// them weighs on the first screen.
import type { Component } from 'svelte'

import { getApp, isAvailable } from '../../apps/registry'
import type { AppId } from '../../apps/types'
import type { Platform } from '../api/servers'
import { t } from '../i18n'
import type { WidgetKind } from './kinds'
import type { WidgetSize } from './layout'

export interface WidgetProps {
  serverID: string
  size: WidgetSize
}

export interface WidgetDef {
  label: () => string
  sizes: readonly WidgetSize[]
  /** The app a click on the widget opens. */
  opens?: AppId
  load: () => Promise<{ default: Component<WidgetProps> }>
}

export const widgetDefs: Record<WidgetKind, WidgetDef> = {
  monitor: {
    label: () => t.apps.monitor,
    sizes: ['S', 'M'],
    opens: 'monitor',
    load: () => import('./MonitorWidget.svelte'),
  },
  docker: {
    label: () => t.apps.docker,
    sizes: ['S', 'M'],
    opens: 'docker',
    load: () => import('./DockerWidget.svelte'),
  },
  services: {
    label: () => t.apps.services,
    sizes: ['S', 'M'],
    opens: 'services',
    load: () => import('./ServicesWidget.svelte'),
  },
}

/** Does this kind of widget work on a server running os: the app it opens does. */
export function widgetAvailable(kind: WidgetKind, os: Platform | undefined): boolean {
  const opens = widgetDefs[kind].opens
  return opens ? isAvailable(getApp(opens), os) : true
}
