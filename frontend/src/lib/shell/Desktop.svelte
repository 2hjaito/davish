<script lang="ts">
  import { dockApps, getApp } from '../../apps/registry'
  import type { AppDef } from '../../apps/types'
  import { nginxCheckCerts } from '../api/nginx'
  import { hasRuntime } from '../api/runtime'
  import { audit } from '../stores/audit.svelte'
  import { notifications } from '../stores/notifications.svelte'
  import { servers } from '../stores/servers.svelte'
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
  import { openFilesAt, openLogs, openTransfers } from '../wm/intents.svelte'
  import Banners from './Banners.svelte'
  import { ui } from '../stores/ui.svelte'
  import Window from '../wm/Window.svelte'
  import { wm } from '../wm/windowManager.svelte'
  import Dock from './Dock.svelte'
  import MenuBar from './MenuBar.svelte'
  import HostKeyDialog from './HostKeyDialog.svelte'
  import ServerDialog from './ServerDialog.svelte'
  import SleepScreen from './SleepScreen.svelte'
  import { fadeOut } from '../motion'
  import { hasPrimaryModifier, isMac } from './platform'

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
    notifications.init((id) => servers.list.find((s) => s.id === id)?.name ?? id, openLogs),
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
    }
  })

  // Each server reopens the windows it had (docs/ARCHITECTURE.md §5.1).
  wm.keepInLayout((appId) => getApp(appId)?.layout !== false)
  function openOptions(appId: string) {
    const app = getApp(appId)
    return app && { title: app.label, size: app.size }
  }

  // First run: ask for a server. Otherwise the active server comes back as
  // it was left; one never seen before opens a terminal, like the mockup's
  // Terminal + Files in the browser preview.
  let opened = false
  $effect(() => {
    if (!servers.ready || opened) return
    opened = true
    if (servers.list.length === 0) {
      ui.addServer()
      return
    }
    const id = servers.activeId
    restored.add(id)
    if (wm.hasLayout(id)) {
      wm.restoreLayout(id, openOptions)
      return
    }
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
  })

  // Switching to a workspace for the first time this session brings back
  // its windows.
  // eslint-disable-next-line svelte/prefer-svelte-reactivity -- bookkeeping, never rendered
  const restored = new Set<string>()
  $effect(() => {
    const id = servers.activeId
    if (!opened || !id || restored.has(id)) return
    restored.add(id)
    wm.restoreLayout(id, openOptions)
  })

  function onKeyDown(e: KeyboardEvent) {
    if (!hasPrimaryModifier(e) || e.altKey || e.shiftKey) return
    // Off macOS the modifier is Ctrl, which the terminal needs (Ctrl+W, Ctrl+1…).
    if (!isMac && e.target instanceof Element && e.target.closest('.xterm')) return
    const key = e.key.toLowerCase()
    if (key === 'w' && focused) {
      wm.close(focused.id)
    } else if (key === 'm' && focused) {
      wm.minimize(focused.id)
    } else if (key === ',') {
      const settings = getApp('settings')
      if (settings) launch(settings)
    } else if (/^[1-9]$/.test(key)) {
      servers.selectIndex(Number(key) - 1)
    } else {
      return
    }
    e.preventDefault()
  }
</script>

<svelte:window bind:innerWidth bind:innerHeight onkeydown={onKeyDown} />

<div class="desktop">
  <MenuBar />

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
          <app.component {win} />
        </Window>
      </div>
    {/if}
  {/each}

  <Dock
    apps={dockApps}
    isOpen={(id) => wm.isOpen(id, serverID)}
    onlaunch={launch}
    bind:height={dockHeight}
  />

  <Banners />
  <ServerDialog />
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
</style>
