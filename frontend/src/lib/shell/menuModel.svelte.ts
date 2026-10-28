// What the menu bar holds, built from the current state each time it is read.
import { dockApps, getApp } from '../../apps/registry'
import { t } from '../i18n/vi'
import { appLock } from '../stores/lock.svelte'
import { servers } from '../stores/servers.svelte'
import { ui } from '../stores/ui.svelte'
import { openAudit, openTerminalAt, openTransfers } from '../wm/intents.svelte'
import { wm } from '../wm/windowManager.svelte'
import { appMenus, type Menu, type MenuEntry } from './menus.svelte'

/** Shortcuts of the Go menu, by app. */
export const GO_KEYS: Record<string, string> = {
  terminal: 'shift+T',
  files: 'shift+F',
  monitor: 'shift+M',
  services: 'shift+S',
  docker: 'shift+D',
  logs: 'shift+L',
  editor: 'shift+E',
  git: 'shift+G',
}

export function openApp(appId: string) {
  const app = getApp(appId)
  if (app) wm.openOrFocus(app.id, servers.activeId, { title: app.label, size: app.size })
}

/** The app menu, for the focused window, then File, View, Go, Window and Help. */
export function buildMenus(): Menu[] {
  const m = t.menu
  const server = servers.active
  const id = servers.activeId
  const focused = wm.focused(id)
  const focusedApp = focused && getApp(focused.appId)
  const online = server?.state === 'connected' || server?.state === 'connecting'
  const windows = wm.visible(id)

  const menus: Menu[] = []
  if (focused && focusedApp) {
    const items: MenuEntry[] = [...appMenus.items(focused.id)]
    if (items.length) items.push(null)
    items.push({
      label: m.appHelp(focusedApp.label),
      run: () => (ui.help = { kind: 'app', appId: focusedApp.id }),
    })
    menus.push({ id: 'app', label: focusedApp.label, bold: true, items })
  }

  menus.push(
    {
      id: 'file',
      label: t.menubar.menus.file,
      items: [
        { label: m.addServer, run: () => ui.addServer() },
        { label: m.importServers, run: () => (ui.importDialog = true) },
        null,
        server && online
          ? { label: m.disconnect(server.name), run: () => void servers.disconnect(server.id) }
          : {
              label: m.connect(server?.name ?? ''),
              disabled: !server,
              run: () => server && void servers.reconnect(server.id),
            },
        null,
        { label: m.newTerminal, disabled: !server, run: () => openTerminalAt(id, undefined) },
        {
          label: m.openPath,
          disabled: !server,
          run: () => {
            ui.spotlightSeed = '/'
            ui.spotlight = true
          },
        },
        null,
        {
          label: m.closeWindow,
          combo: 'W',
          disabled: !focused,
          run: () => focused && wm.close(focused.id),
        },
        {
          label: m.lock,
          combo: 'L',
          disabled: !appLock.state.enabled,
          run: () => appLock.lock(),
        },
      ],
    },
    {
      id: 'view',
      label: t.menubar.menus.view,
      items: [
        { label: m.spotlight, combo: 'K', run: () => (ui.spotlight = !ui.spotlight) },
        { label: m.launchpad, combo: 'shift+A', run: () => (ui.launchpad = !ui.launchpad) },
        null,
        {
          label: m.refresh,
          combo: 'R',
          disabled: !focused || !appMenus.refresh(focused.id),
          run: () => focused && appMenus.refresh(focused.id)?.(),
        },
        null,
        { label: m.audit, run: () => openAudit(id) },
        { label: m.transfers, run: () => openTransfers(id) },
      ],
    },
    {
      id: 'go',
      label: t.menubar.menus.go,
      items: [
        ...dockApps.map((a) => ({
          label: a.label,
          combo: GO_KEYS[a.id],
          disabled: !server,
          checked: focused?.appId === a.id,
          run: () => openApp(a.id),
        })),
        null,
        { label: t.apps.settings, combo: ',', run: () => openApp('settings') },
      ],
    },
    {
      id: 'window',
      label: t.menubar.menus.window,
      items: [
        {
          label: m.minimize,
          combo: 'M',
          disabled: !focused,
          run: () => focused && wm.minimize(focused.id),
        },
        { label: m.zoom, disabled: !focused, run: () => focused && wm.toggleMaximize(focused.id) },
        {
          label: m.snapLeft,
          disabled: !focused,
          run: () => focused && wm.snap(focused.id, 'left'),
        },
        {
          label: m.snapRight,
          disabled: !focused,
          run: () => focused && wm.snap(focused.id, 'right'),
        },
        null,
        {
          label: m.bringAll,
          disabled: windows.length === 0,
          run: () => [...windows].sort((a, b) => a.z - b.z).forEach((w) => wm.focus(w.id)),
        },
        {
          label: m.closeAll,
          disabled: windows.length === 0,
          run: () => windows.forEach((w) => wm.close(w.id)),
        },
        null,
        ...(windows.length
          ? windows.map((w) => ({
              label: w.title,
              checked: w.id === focused?.id,
              run: () => wm.focus(w.id),
            }))
          : [{ label: m.noWindows, disabled: true, run: () => {} }]),
      ],
    },
    {
      id: 'help',
      label: t.menubar.menus.help,
      items: [
        { label: m.tour, run: () => (ui.tour = true) },
        { label: m.shortcuts, combo: '/', run: () => (ui.help = { kind: 'shortcuts' }) },
        null,
        { label: m.about, run: () => (ui.help = { kind: 'about' }) },
        { label: m.changelog, run: () => (ui.help = { kind: 'about' }) },
      ],
    },
  )
  return menus
}
