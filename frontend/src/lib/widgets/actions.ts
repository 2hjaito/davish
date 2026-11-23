// What adding a widget and keeping icons clear of widgets decide, apart from drawing.
import { notifications } from '../stores/notifications.svelte'
import { settings } from '../stores/settings.svelte'
import { ui } from '../stores/ui.svelte'
import { widgets } from '../stores/widgets.svelte'
import { t } from '../i18n'
import { type Cell, gridOf } from '../shell/iconGrid'
import { wm } from '../wm/windowManager.svelte'
import type { WidgetKind } from './kinds'
import type { WidgetSize } from './layout'

/** The cells under serverID's widgets, which icons keep clear of; none when widgets are off. */
export function reservedFor(serverID: string, grid = gridOf(wm.area)): Cell[] {
  return settings.knobs.widgets ? widgets.reserved(serverID, grid) : []
}

/** Adds a widget to serverID's desktop and opens editing; says so when there is no room. */
export function addWidget(serverID: string, kind: WidgetKind, size: WidgetSize = 'S'): boolean {
  const made = widgets.add(serverID, kind, size, gridOf(wm.area))
  if (!made) {
    notifications.add({
      id: `widget-room-${Date.now()}`,
      level: 'warn',
      title: t.widgets.noRoom,
      body: t.widgets.noRoomBody,
      serverID,
      at: Date.now(),
    })
    return false
  }
  ui.widgetEdit = true
  return true
}
