<script lang="ts">
  import { formatClock, formatGB, formatRate } from '../format'
  import { t } from '../i18n/vi'
  import { metrics } from '../stores/metrics.svelte'
  import { servers } from '../stores/servers.svelte'
  import { ui } from '../stores/ui.svelte'
  import { level } from '../ui/charts/scale'
  import Icon from '../ui/Icon.svelte'
  import MenuDropdown from './MenuDropdown.svelte'
  import { buildMenus } from './menuModel.svelte'
  import NotificationMenu from './NotificationMenu.svelte'
  import { hasNativeTrafficLights, shortcut } from './platform'
  import ServerPicker from './ServerPicker.svelte'
  import SystemMenu from './SystemMenu.svelte'

  let now = $state(new Date())
  const menus = $derived(buildMenus())
  let openId = $state('')

  function step(dir: -1 | 1) {
    const i = menus.findIndex((m) => m.id === openId)
    openId = menus[(i + dir + menus.length) % menus.length]?.id ?? ''
  }
  // Only a connected server is watched, so a reconnect starts a fresh
  // collector and the numbers disappear while the server is unreachable.
  const watchedId = $derived(servers.active?.state === 'connected' ? servers.active.id : '')
  const m = $derived(watchedId ? metrics.latest(watchedId) : undefined)
  const memPct = $derived(m && m.memTotal > 0 ? (m.memUsed / m.memTotal) * 100 : 0)

  $effect(() => {
    const timer = setInterval(() => (now = new Date()), 1000)
    return () => clearInterval(timer)
  })

  $effect(() => {
    if (watchedId) return metrics.watch(watchedId)
  })
</script>

<header class="menubar glass" class:native-lights={hasNativeTrafficLights}>
  <nav class="left" aria-label={t.menubar.label}>
    <SystemMenu />
    <ServerPicker />
    {#each menus as menu (menu.id)}
      <MenuDropdown
        {menu}
        open={openId === menu.id}
        onopen={() => (openId = menu.id)}
        onhover={() => openId && (openId = menu.id)}
        onclose={() => openId === menu.id && (openId = '')}
        onstep={step}
      />
    {/each}
  </nav>

  <div class="right">
    {#if m}
      <span class="stat {level(m.cpu, 85, 95)}">
        <span class="label">{t.menubar.cpu}</span>
        <span class="bar"><span style:width="{m.cpu}%"></span></span>
        <span>{Math.round(m.cpu)}%</span>
      </span>
      <span class="stat {level(memPct, 90, 95)}">
        <span class="label">{t.menubar.ram}</span>
        <span>{formatGB(m.memUsed)} / {formatGB(m.memTotal)} GB</span>
      </span>
      <span class="stat {level(m.diskPct, 85, 95)}">
        <span class="label">{t.menubar.disk}</span>
        <span>{m.diskPct}%</span>
      </span>
      <span class="stat net" aria-label={t.menubar.network}>
        <Icon name="network" size={13} strokeWidth={2.2} />
        <span>{formatRate(m.rxBps)}</span>
      </span>
    {/if}
    <button
      class="mbtn search"
      aria-label={t.menubar.search(shortcut('K'))}
      title={t.menubar.search(shortcut('K'))}
      aria-expanded={ui.spotlight}
      onclick={() => ui.toggleSpotlight()}
    >
      <Icon name="search" size={15} />
    </button>
    <NotificationMenu />
    <span class="clock">{formatClock(now, t.menubar.weekdays)}</span>
  </div>
</header>

<style>
  .menubar {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    height: var(--menubar-h);
    z-index: 1000;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 12px;
    font-size: var(--text-md);
    border-radius: 0;
    border-width: 0 0 1px 0;
    /* Lets the user move the OS window by the menu bar when it is not maximized. */
    --wails-draggable: drag;
  }

  /*
   * The blur sits on a pseudo-element: an element with backdrop-filter becomes
   * the backdrop root of its descendants, which would leave the dropdown menus
   * inside the bar unable to blur the windows underneath them.
   */
  .menubar.glass {
    background: none;
    -webkit-backdrop-filter: none;
    backdrop-filter: none;
  }

  .menubar::before {
    content: '';
    position: absolute;
    inset: 0;
    z-index: -1;
    background: var(--glass-bg);
    -webkit-backdrop-filter: var(--glass-blur);
    backdrop-filter: var(--glass-blur);
  }

  /* Room for the macOS close/minimize/zoom buttons drawn by the OS. */
  .menubar.native-lights {
    padding-left: 78px;
  }

  .left,
  .right {
    display: flex;
    align-items: center;
  }

  .left {
    gap: 2px;
  }

  .right {
    gap: 16px;
    font-variant-numeric: tabular-nums;
  }

  /* Custom properties inherit, so reset it for everything inside the bar,
     otherwise clicks on buttons and menus would start a window drag. */
  .menubar :global(*) {
    --wails-draggable: no-drag;
  }

  .menubar :global(.mbtn) {
    background: transparent;
    border: 0;
    padding: 2px 9px;
    border-radius: 6px;
    transition: background var(--dur-fast) var(--ease-out);
  }

  .menubar :global(.mbtn:hover) {
    background: var(--glass-hover);
  }

  .menubar :global(.mbtn.picker) {
    background: var(--glass-highlight);
  }

  .stat {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .stat.net {
    gap: 4px;
  }

  .label {
    font-size: var(--text-xs);
    color: var(--fg-label);
  }

  .bar {
    width: 34px;
    height: 6px;
    border-radius: 3px;
    background: var(--fill-track);
    overflow: hidden;
  }

  .bar span {
    display: block;
    height: 100%;
    background: var(--fg);
    transition: width var(--dur-slow) var(--ease-out);
  }

  .warn {
    color: var(--warn);
  }
  .warn .bar span {
    background: var(--warn);
  }
  .danger {
    color: var(--danger);
  }
  .danger .bar span {
    background: var(--danger);
  }

  .clock {
    font-weight: 500;
  }
</style>
