<script lang="ts">
  import { untrack } from 'svelte'

  import { apps, getApp, loadApp } from '../../apps/registry'
  import type { AppDef } from '../../apps/types'
  import { monitorSetIdle, monitorSetThresholds } from '../api/monitor'
  import { nginxCheckCerts } from '../api/nginx'
  import { packagesCheckSecurity } from '../api/packages'
  import { exitDemo, hasRuntime, isDemo } from '../api/runtime'
  import { audit } from '../stores/audit.svelte'
  import { dockPins } from '../stores/dock.svelte'
  import { appLock } from '../stores/lock.svelte'
  import { notifications } from '../stores/notifications.svelte'
  import { servers } from '../stores/servers.svelte'
  import { settings } from '../stores/settings.svelte'
  import { sleep } from '../stores/sleep.svelte'
  import { toUserError } from '../api/errors'
  import { transferReveal } from '../api/transfer'
  import { t } from '../i18n/vi'
  import {
    describeTransfer,
    type Transfer,
    type TransferActions,
    transferNoticeId,
    transfers,
  } from '../stores/transfers.svelte'
  import { openDisk, openFilesAt, openLogs, openTransfers } from '../wm/intents.svelte'
  import Banners from './Banners.svelte'
  import { ui } from '../stores/ui.svelte'
  import Window from '../wm/Window.svelte'
  import { wm } from '../wm/windowManager.svelte'
  import Spotlight from '../spotlight/Spotlight.svelte'
  import Dock from './Dock.svelte'
  import { dockItems } from './dockItems'
  import Launchpad from './Launchpad.svelte'
  import Welcome from './Welcome.svelte'
  import MenuBar from './MenuBar.svelte'
  import HostKeyDialog from './HostKeyDialog.svelte'
  import ServerDialog from './ServerDialog.svelte'
  import ImportDialog from './ImportDialog.svelte'
  import HelpDialog from './HelpDialog.svelte'
  import Tour from './Tour.svelte'
  import UpdateDialog from './UpdateDialog.svelte'
  import { updates } from '../stores/updates.svelte'
  import SleepScreen from './SleepScreen.svelte'
  import { fadeOut } from '../motion'
  import { hasPrimaryModifier, isMac } from './platform'
  import { buildMenus } from './menuModel.svelte'
  import { findShortcut } from './menus.svelte'

  const MENUBAR_H = 30
  const DOCK_BOTTOM = 14

  let innerWidth = $state(window.innerWidth)
  let innerHeight = $state(window.innerHeight)
  let dockHeight = $state(80)

  const serverID = $derived(servers.activeId)
  const focused = $derived(wm.focused(serverID))

  $effect(() => {
    wm.area = {
      x: 0,
      y: MENUBAR_H,
      w: innerWidth,
      h: Math.max(200, innerHeight - MENUBAR_H - dockHeight - DOCK_BOTTOM - 6),
    }
  })

  function launch(app: AppDef): boolean {
    return wm.openOrFocus(app.id, serverID, { title: app.label, size: app.size })
  }

  const dockTiles = $derived([
    { id: 'launchpad' as const, label: t.launchpad.label },
    ...dockItems(apps, dockPins.ids, (id) => wm.isOpen(id, serverID)),
  ])

  function launchTile(id: AppDef['id'] | 'launchpad'): boolean {
    if (id === 'launchpad') {
      ui.launchpad = !ui.launchpad
      return false
    }
    const app = getApp(id)
    return app ? launch(app) : false
  }

  // Load saved servers and follow connection events for the app's lifetime.
  $effect(() => {
    let stop = () => {}
    let disposed = false
    void servers.init().then((off) => (disposed ? off() : (stop = off)))
    return () => {
      disposed = true
      stop()
    }
  })

  // Threshold alerts and other notify:new events, for the app's lifetime.
  $effect(() =>
    notifications.init(
      (id) => servers.list.find((s) => s.id === id)?.name ?? id,
      openLogs,
      openDisk,
    ),
  )

  // Transfers outlive the Files window that started them: a download says
  // when it starts and how it ends, in one notification per job.
  $effect(() => {
    const actions: TransferActions = {
      showProgress: () => openTransfers(serverID),
      showCopy: (j) => openFilesAt(j.serverID, j.dst.slice(0, j.dst.lastIndexOf('/')) || '/'),
      nameOf: (id) => servers.list.find((s) => s.id === id)?.name ?? id,
      reveal: (j) =>
        void transferReveal(j.dst).catch((e: unknown) =>
          notifications.add({
            id: `reveal-${j.id}`,
            level: 'error',
            title: t.transfers.reveal,
            body: toUserError(e).message,
            serverID: j.serverID,
            at: Date.now(),
          }),
        ),
    }
    const notify = (j: Transfer) => {
      const n = describeTransfer(j, actions)
      if (n) notifications.add(n)
      else notifications.dismiss(transferNoticeId(j))
    }
    return transfers.init(notify, notify)
  })

  $effect(() => audit.init())
  $effect(() => updates.init())

  // Settings that reach outside the frontend's own components: the mono
  // font of every view, and the thresholds sysmon notifies at.
  $effect(() => {
    document.documentElement.style.setProperty('--font-mono', settings.monoFamily)
  })
  $effect(() => {
    if (!hasRuntime) return
    void monitorSetThresholds(settings.alertMem, settings.alertDisk, settings.alertHold)
  })

  // Nobody looks at Davish while it is hidden, minimized, locked or asleep:
  // metrics are then read every 30s instead of every 2s, enough for alerts.
  let hidden = $state(document.hidden)
  $effect(() => {
    if (!hasRuntime) return
    void monitorSetIdle(hidden || appLock.state.locked || sleep.active)
  })

  // TLS certificates nearing expiry notify once a server connects; checked
  // once per server for the app's lifetime (Go keeps what it notified).
  // eslint-disable-next-line svelte/prefer-svelte-reactivity -- bookkeeping, never rendered
  const certsChecked = new Set<string>()
  $effect(() => {
    if (!hasRuntime) return
    for (const s of servers.list) {
      if (s.state !== 'connected' || certsChecked.has(s.id)) continue
      certsChecked.add(s.id)
      void nginxCheckCerts(s.id).catch(() => {})
      void packagesCheckSecurity(s.id).catch(() => {})
    }
  })

  // Each server reopens the windows it had (docs/ARCHITECTURE.md §5.1).
  wm.keepInLayout((appId) => getApp(appId)?.layout !== false)
  function openOptions(appId: string) {
    const app = getApp(appId)
    return app && { title: app.label, size: app.size }
  }

  /**
   * A workspace showing for the first time this session comes back as it
   * was left; one never seen before opens a terminal (and, in the browser
   * preview, Files, like the mockup).
   */
  function showWorkspace(id: string) {
    if (wm.hasLayout(id)) {
      wm.restoreLayout(id, openOptions)
      return
    }
    if (wm.visible(id).length > 0) return
    const start = hasRuntime
      ? ([['terminal', { x: 120, y: 120 }]] as const)
      : ([
          ['terminal', { x: 120, y: 120 }],
          ['files', { x: 790, y: 200 }],
        ] as const)
    for (const [appId, at] of start) {
      const app = getApp(appId)
      if (app) wm.open(app.id, id, { title: app.label, size: app.size, at })
    }
  }

  // Once servers are known, and on every switch (including to a server just
  // added): with no server yet, the welcome page asks for one.
  // eslint-disable-next-line svelte/prefer-svelte-reactivity -- bookkeeping, never rendered
  const shown = new Set<string>()
  $effect(() => {
    const id = servers.activeId
    if (!servers.ready || !id || !servers.list.some((s) => s.id === id) || shown.has(id)) return
    shown.add(id)
    untrack(() => showWorkspace(id))
  })

  // Locks after settings.lockAfter idle minutes. Activity is any key or
  // pointer use; checked every 15s, which is precise enough for minutes.
  let lastActive = Date.now()
  const markActive = () => (lastActive = Date.now())
  $effect(() => {
    const timer = setInterval(() => {
      const after = settings.lockAfter
      if (!appLock.state.enabled || appLock.state.locked || after === 0) {
        lastActive = Date.now()
        return
      }
      if (Date.now() - lastActive > after * 60_000) appLock.lock()
    }, 15_000)
    return () => clearInterval(timer)
  })

  // Every icon-only button gets its accessible name as a tooltip, so none
  // is left unexplained; buttons with their own title, or their own
  // tooltip like the Dock's label, keep it.
  function tooltip(e: PointerEvent) {
    const b =
      e.target instanceof Element
        ? e.target.closest('button[aria-label]:not([title]):not([data-own-tooltip])')
        : null
    const label = b?.getAttribute('aria-label')
    if (b && label && !b.textContent?.trim()) b.setAttribute('title', label)
  }

  function onKeyDown(e: KeyboardEvent) {
    if (!hasPrimaryModifier(e) || e.altKey) return
    // Off macOS the modifier is Ctrl, which the terminal needs (Ctrl+W,
    // Ctrl+K, Ctrl+1…): there Spotlight opens from outside the terminal.
    if (!isMac && e.target instanceof Element && e.target.closest('.xterm')) return
    if (/^[1-9]$/.test(e.key) && !e.shiftKey) {
      servers.selectIndex(Number(e.key) - 1)
    } else {
      // Every other shortcut is a menu item's (menuModel.svelte.ts).
      const item = findShortcut(buildMenus(), e)
      if (!item) return
      item.run()
    }
    e.preventDefault()
  }
</script>

<svelte:window
  bind:innerWidth
  bind:innerHeight
  onkeydown={onKeyDown}
  onkeydowncapture={markActive}
  onpointerdowncapture={markActive}
  onpointermove={markActive}
  onpointerover={tooltip}
/>
<svelte:document onvisibilitychange={() => (hidden = document.hidden)} />

<div class="desktop">
  <MenuBar />
  <Spotlight open={ui.spotlight} onclose={() => (ui.spotlight = false)} />
  <Launchpad open={ui.launchpad} onclose={() => (ui.launchpad = false)} onlaunch={launch} />

  {#if wm.snapPreview}
    {@const p = wm.snapPreview}
    <div
      class="snap-preview"
      out:fadeOut
      style:transform="translate({p.x}px, {p.y}px)"
      style:width="{p.w}px"
      style:height="{p.h}px"
    ></div>
  {/if}

  {#each wm.windows as win (win.id)}
    {@const app = getApp(win.appId)}
    {#if app}
      <!-- Other workspaces stay mounted so their state survives switching. -->
      <div class="workspace" class:hidden={win.serverID !== serverID}>
        <Window {win} theme={app.theme} focused={focused?.id === win.id}>
          {#await loadApp(app) then AppView}
            <AppView {win} />
          {/await}
        </Window>
      </div>
    {/if}
  {/each}

  {#if servers.ready && servers.list.length === 0}
    <Welcome onadd={() => ui.addServer()} onimport={() => (ui.importDialog = true)} />
  {:else}
    <Dock
      apps={dockTiles}
      isOpen={(id) => wm.isOpen(id, serverID)}
      onlaunch={launchTile}
      onclose={(id) =>
        wm
          .visible(serverID)
          .filter((w) => w.appId === id)
          .forEach((w) => wm.close(w.id))}
      bind:height={dockHeight}
    />
  {/if}

  <Banners />
  <ServerDialog />
  <ImportDialog />
  <HelpDialog />
  <Tour />
  <UpdateDialog />
  {#if isDemo}
    <div class="demo glass glass-strong" role="status">
      <span>{t.demo.banner}</span>
      <button type="button" class="btn" onclick={exitDemo}>{t.demo.exit}</button>
    </div>
  {/if}
  <HostKeyDialog />
  <SleepScreen />
</div>

<style>
  .desktop {
    position: fixed;
    inset: 0;
    /* clip, not hidden: focusing something in a window that hangs off
       screen must not scroll the whole desktop sideways. */
    overflow: clip;
    background-color: var(--wallpaper-base);
    background-image:
      radial-gradient(circle at 18% 22%, var(--wallpaper-accent) 0, transparent 38%),
      radial-gradient(circle at 82% 30%, var(--wallpaper-pink) 0, transparent 34%),
      radial-gradient(circle at 60% 88%, var(--wallpaper-teal) 0, transparent 40%),
      radial-gradient(circle at 30% 80%, var(--wallpaper-gold) 0, transparent 30%);
  }

  .workspace.hidden {
    display: none;
  }

  .snap-preview {
    position: absolute;
    top: 0;
    left: 0;
    z-index: 800;
    border-radius: var(--window-radius);
    background: var(--snap-preview);
    border: 1px solid var(--snap-preview-border);
    pointer-events: none;
    transition:
      transform var(--dur) var(--ease-out),
      width var(--dur) var(--ease-out),
      height var(--dur) var(--ease-out);
    animation: appear var(--dur) var(--ease-out);
  }

  @keyframes appear {
    from {
      opacity: 0;
      scale: 0.97;
    }
  }

  .demo {
    position: fixed;
    top: calc(var(--menubar-h) + 8px);
    left: 50%;
    translate: -50% 0;
    z-index: 900;
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 6px 8px 6px 14px;
    border-radius: 12px;
    font-size: var(--text-sm);
  }
</style>
