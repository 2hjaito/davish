<script lang="ts">
  import { apps, getApp } from '../../apps/registry'
  import type { AppDef } from '../../apps/types'
  import { servers } from '../stores/servers.svelte'
  import Window from '../wm/Window.svelte'
  import { wm } from '../wm/windowManager.svelte'
  import Dock from './Dock.svelte'
  import MenuBar from './MenuBar.svelte'
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

  // Start like the mockup: Terminal and Files open on the first server.
  const first = servers.list[0]
  if (first && wm.windows.length === 0) {
    for (const [id, at] of [
      ['terminal', { x: 120, y: 120 }],
      ['files', { x: 790, y: 200 }],
    ] as const) {
      const app = getApp(id)
      if (app) wm.open(app.id, first.id, { title: app.label, size: app.size, at })
    }
  }

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
    {apps}
    isOpen={(id) => wm.isOpen(id, serverID)}
    onlaunch={launch}
    bind:height={dockHeight}
  />

  <SleepScreen />
</div>

<style>
  .desktop {
    position: fixed;
    inset: 0;
    overflow: hidden;
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
