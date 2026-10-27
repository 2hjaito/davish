<script lang="ts">
  import { formatPct, formatRate } from '../../lib/format'
  import { t } from '../../lib/i18n/vi'
  import { metrics, type Series } from '../../lib/stores/metrics.svelte'
  import { load, save } from '../../lib/stores/persist'
  import { servers } from '../../lib/stores/servers.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import type { IconName } from '../../lib/ui/icons'
  import { getWindowContext } from '../../lib/wm/context'
  import TitleBar from '../../lib/wm/TitleBar.svelte'
  import type { AppProps } from '../types'
  import CpuView from './CpuView.svelte'
  import DiskView from './DiskView.svelte'
  import Empty from './Empty.svelte'
  import GpuView from './GpuView.svelte'
  import InfoView from './InfoView.svelte'
  import MemoryView from './MemoryView.svelte'
  import NetworkView from './NetworkView.svelte'
  import OverviewView from './OverviewView.svelte'
  import ProcessesView from './ProcessesView.svelte'
  import './table.css'
  import { memPct, RANGES, type RangeKey, type Section, sectionLabels } from './types'

  let { win }: AppProps = $props()

  const ctx = getWindowContext()

  const saved = load('monitor', { section: 'overview' as Section, range: '5m' as RangeKey })
  let section = $state<Section>(saved.section in sectionLabels ? saved.section : 'overview')
  let range = $state<RangeKey>(saved.range in RANGES ? saved.range : '5m')
  /** The data on screen while paused; collection itself goes on. */
  let frozen = $state.raw<Series | null>(null)

  $effect(() => save('monitor', { section, range }))

  const server = $derived(servers.list.find((s) => s.id === win.serverID))
  const connected = $derived(server?.state === 'connected')

  $effect(() => {
    if (connected) return metrics.watch(win.serverID)
  })

  const live = $derived(metrics.byServer[win.serverID])
  const data = $derived(frozen ?? live)
  const m = $derived(data?.latest)
  const charts = $derived(section !== 'processes' && section !== 'info')

  function togglePause() {
    frozen = frozen ? null : (live ?? null)
  }

  interface Item {
    id: Section
    icon: IconName
    value?: string
  }

  const groups = $derived<{ label: string; items: Item[] }[]>([
    {
      label: server?.name ?? '',
      items: [
        { id: 'overview', icon: 'grid' },
        { id: 'processes', icon: 'list' },
      ],
    },
    {
      label: t.monitor.hardware,
      items: [
        { id: 'cpu', icon: 'cpu', value: m && formatPct(m.cpu) },
        { id: 'memory', icon: 'memory', value: m && formatPct(memPct(m.memUsed, m.memTotal)) },
        {
          id: 'gpu',
          icon: 'gpu',
          value: m?.gpus[0] && m.gpus[0].util >= 0 ? formatPct(m.gpus[0].util) : undefined,
        },
        { id: 'disk', icon: 'disk', value: m && formatRate(m.readBps + m.writeBps) },
        { id: 'network', icon: 'network', value: m && formatRate(m.rxBps + m.txBps) },
      ],
    },
    { label: t.monitor.system, items: [{ id: 'info', icon: 'info' }] },
  ])
</script>

<div class="monitor" class:blurred={!ctx.focused}>
  <nav class="sidebar" aria-label={t.monitor.sections}>
    <TitleBar bare />
    <div class="items">
      {#each groups as g (g.label)}
        <span class="group">{g.label}</span>
        {#each g.items as it (it.id)}
          <button
            class="item"
            class:active={section === it.id}
            aria-current={section === it.id ? 'page' : undefined}
            onclick={() => (section = it.id)}
          >
            <Icon name={it.icon} size={16} />
            <span class="label">{sectionLabels[it.id]}</span>
            {#if it.value}<span class="value">{it.value}</span>{/if}
          </button>
        {/each}
      {/each}
    </div>
  </nav>

  <div class="main">
    <!-- Pointer handlers here only add window dragging, as on the title bar. -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="toolbar" onpointerdown={ctx.startDrag} ondblclick={ctx.toggleMaximize}>
      <h2 class="title">{sectionLabels[section]}</h2>
      {#if frozen}<span class="paused">{t.monitor.paused}</span>{/if}
      <span class="spacer"></span>
      {#if charts}
        <div class="segmented" role="group" aria-label={t.monitor.range}>
          {#each Object.keys(RANGES) as RangeKey[] as r (r)}
            <button
              aria-pressed={range === r}
              aria-label={t.monitor.ranges[r]}
              onclick={() => (range = r)}>{t.monitor.rangeShort[r]}</button
            >
          {/each}
        </div>
      {/if}
      <button
        class="pause"
        aria-pressed={frozen !== null}
        aria-label={frozen ? t.monitor.resume : t.monitor.pause}
        title={frozen ? t.monitor.resume : t.monitor.pause}
        disabled={!frozen && !live}
        onclick={togglePause}
      >
        <Icon name={frozen ? 'play' : 'pause'} size={13} strokeWidth={2.4} />
      </button>
    </div>

    <div class="content" class:fill={section === 'processes'}>
      {#if !connected && !frozen}
        <Empty icon="server" title={t.monitor.offline} hint={t.monitor.offlineHint} />
      {:else if section === 'processes'}
        <ProcessesView serverID={win.serverID} paused={frozen !== null} />
      {:else if !data}
        <Empty icon="monitor" title={t.monitor.waiting} />
      {:else if section === 'info'}
        <InfoView serverID={win.serverID} latest={data.latest} />
      {:else}
        {@const props = { data, range: RANGES[range], rangeLabel: t.monitor.ranges[range] }}
        {#if section === 'overview'}
          <OverviewView {...props} open={(s) => (section = s)} />
        {:else if section === 'cpu'}
          <CpuView {...props} />
        {:else if section === 'memory'}
          <MemoryView {...props} />
        {:else if section === 'gpu'}
          <GpuView {...props} />
        {:else if section === 'disk'}
          <DiskView {...props} />
        {:else if section === 'network'}
          <NetworkView {...props} />
        {/if}
      {/if}
    </div>
  </div>
</div>

<style>
  /* Same Finder layout as Files: a translucent sidebar beside an almost
     opaque pane, so numbers read cleanly over any wallpaper. */
  .monitor {
    flex: 1;
    min-height: 0;
    display: flex;
  }

  .sidebar {
    width: var(--sidebar-w);
    flex-shrink: 0;
    display: flex;
    flex-direction: column;
  }

  .items {
    flex: 1;
    min-height: 0;
    padding: 4px 10px 10px;
    display: flex;
    flex-direction: column;
    gap: 1px;
    overflow-y: auto;
  }

  .group {
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--fg-subtle);
    padding: 6px 8px 4px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .group:not(:first-child) {
    padding-top: 14px;
  }

  .item {
    display: flex;
    align-items: center;
    gap: 8px;
    text-align: left;
    border: 0;
    background: none;
    padding: 5px 8px;
    border-radius: var(--control-radius);
    transition: background var(--dur-fast) var(--ease-out);
  }

  .item :global(svg) {
    color: var(--accent);
    flex-shrink: 0;
  }

  .item:hover:not(.active) {
    background: var(--row-hover);
  }

  .item.active {
    background: var(--sidebar-active);
  }

  .label {
    flex: 1;
    min-width: 0;
  }

  .value {
    font-size: var(--text-xs);
    color: var(--fg-muted);
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
  }

  .main {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    background: var(--content-bg);
    border-left: 1px solid var(--divider-soft);
  }

  .toolbar {
    height: var(--window-titlebar-h);
    flex-shrink: 0;
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 0 10px 0 16px;
    border-bottom: 1px solid var(--divider-soft);
  }

  .title {
    margin: 0;
    font-size: var(--text-lg);
    font-weight: 600;
    white-space: nowrap;
  }

  .paused {
    padding: 1px 8px;
    border-radius: 10px;
    background: var(--fill-soft);
    font-size: var(--text-xs);
    color: var(--fg-muted);
  }

  .spacer {
    flex: 1;
  }

  .segmented {
    display: flex;
    padding: 2px;
    border-radius: 8px;
    background: var(--fill-soft);
  }

  .segmented button {
    min-width: 36px;
    padding: 2px 8px;
    border: 0;
    border-radius: 6px;
    background: none;
    font-size: var(--text-xs);
    color: var(--fg-muted);
    transition: background var(--dur-fast) var(--ease-out);
  }

  .segmented button[aria-pressed='true'] {
    background: var(--chart-surface);
    color: var(--fg);
    font-weight: 600;
    box-shadow: var(--switch-knob-shadow);
  }

  .pause {
    display: flex;
    padding: 6px;
    border: 0;
    border-radius: 50%;
    background: var(--fill-soft);
    color: var(--fg-muted);
    transition: background var(--dur-fast) var(--ease-out);
  }

  .pause:hover:not(:disabled) {
    background: var(--fill-track);
  }

  .pause[aria-pressed='true'] {
    background: var(--accent);
    color: var(--fg-on-accent);
  }

  .pause:disabled {
    opacity: 0.5;
  }

  .content {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    padding: 14px 16px 16px;
    overflow-y: auto;
  }

  /* The process table scrolls by itself under a fixed search bar. */
  .content.fill {
    overflow: hidden;
  }
</style>
