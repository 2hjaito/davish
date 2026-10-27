<script lang="ts">
  import { untrack } from 'svelte'

  import { toUserError } from '../../lib/api/errors'
  import type { Unit } from '../../lib/api/services'
  import { t } from '../../lib/i18n/vi'
  import { load, save } from '../../lib/stores/persist'
  import { servers } from '../../lib/stores/servers.svelte'
  import Empty from '../../lib/ui/Empty.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import type { IconName } from '../../lib/ui/icons'
  import { getWindowContext } from '../../lib/wm/context'
  import TitleBar from '../../lib/wm/TitleBar.svelte'
  import type { AppProps } from '../types'
  import { backend } from './backend'
  import ServiceInspector from './ServiceInspector.svelte'
  import ServiceList from './ServiceList.svelte'
  import { counts, type Filter, FILTERS, visible } from './units'

  let { win }: AppProps = $props()

  const ctx = getWindowContext()

  const saved = load('services', { filter: 'all' as Filter })
  let filter = $state<Filter>(FILTERS.includes(saved.filter) ? saved.filter : 'all')
  let query = $state('')
  let units = $state.raw<Unit[] | null>(null)
  let error = $state('')
  let loading = $state(false)
  let selected = $state<string | null>(null)

  $effect(() => save('services', { filter }))

  const server = $derived(servers.list.find((s) => s.id === win.serverID))
  const connected = $derived(server?.state === 'connected')
  const shown = $derived(units ? visible(units, filter, query) : [])
  const count = $derived(units ? counts(units) : null)
  const current = $derived(units?.find((u) => u.name === selected) ?? null)

  const icons: Record<Filter, IconName> = {
    all: 'services',
    running: 'play',
    failed: 'warning',
    stopped: 'pause',
  }

  async function refresh() {
    if (loading) return
    loading = true
    try {
      units = await backend.list(win.serverID)
      error = ''
    } catch (e) {
      error = toUserError(e).message
    } finally {
      loading = false
    }
  }

  // Load when the server comes up, and again whenever the window regains
  // focus, since services change behind our back. There is no polling: every
  // list is a command in the audit log.
  // refresh reads state of its own, which must not become a dependency.
  $effect(() => {
    if (connected) untrack(() => void refresh())
  })
  let wasFocused = untrack(() => ctx.focused)
  $effect(() => {
    const f = ctx.focused
    if (f && !wasFocused && connected) untrack(() => void refresh())
    wasFocused = f
  })
</script>

<div class="services" class:blurred={!ctx.focused}>
  <nav class="sidebar" aria-label={t.services.sections}>
    <TitleBar bare />
    <div class="items">
      <span class="group">{server?.name ?? ''}</span>
      {#each FILTERS as f (f)}
        <button
          class="item"
          class:active={filter === f}
          aria-current={filter === f ? 'page' : undefined}
          onclick={() => (filter = f)}
        >
          <Icon name={icons[f]} size={16} />
          <span class="label">{t.services.filters[f]}</span>
          {#if count}
            <span class="value" class:bad={f === 'failed' && count.failed > 0}>{count[f]}</span>
          {/if}
        </button>
      {/each}
    </div>
  </nav>

  <div class="main">
    <!-- Pointer handlers here only add window dragging, as on the title bar. -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="toolbar" onpointerdown={ctx.startDrag} ondblclick={ctx.toggleMaximize}>
      <h2 class="title">{t.services.filters[filter]}</h2>
      <span class="spacer"></span>
      <label class="search" onpointerdown={(e) => e.stopPropagation()}>
        <Icon name="search" size={13} strokeWidth={2.2} />
        <input
          type="text"
          placeholder={t.services.search}
          aria-label={t.services.search}
          bind:value={query}
          onkeydown={(e) => e.key === 'Escape' && (query = '')}
        />
      </label>
      <button
        class="tool"
        aria-label={t.services.refresh}
        title={t.services.refresh}
        disabled={!connected || loading}
        onpointerdown={(e) => e.stopPropagation()}
        onclick={() => void refresh()}
      >
        <Icon name="refresh" size={14} strokeWidth={2.2} />
      </button>
    </div>

    {#if !connected}
      <Empty icon="server" title={t.services.offline} hint={t.services.offlineHint} />
    {:else if !units}
      <Empty icon={error ? 'warning' : 'services'} title={error || t.services.loading} />
    {:else}
      <div class="split">
        <div class="list">
          {#if error}<p class="error"><Icon name="warning" size={13} /> {error}</p>{/if}
          <ServiceList units={shown} {selected} onselect={(n) => (selected = n)} />
          <p class="footer">{t.services.count(shown.length)}</p>
        </div>
        <div class="detail">
          {#if current}
            <ServiceInspector
              serverID={win.serverID}
              unit={current}
              onchanged={() => void refresh()}
            />
          {:else}
            <Empty icon="services" title={t.services.noSelection} />
          {/if}
        </div>
      </div>
    {/if}
  </div>
</div>

<style>
  /* Finder layout, as in Monitor: a translucent sidebar of filters beside
     an almost opaque pane. */
  .services {
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
  }

  .value.bad {
    color: var(--danger);
    font-weight: 600;
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
    gap: 8px;
    padding: 0 10px 0 16px;
    border-bottom: 1px solid var(--divider-soft);
  }

  .title {
    margin: 0;
    font-size: var(--text-lg);
    font-weight: 600;
    white-space: nowrap;
  }

  .spacer {
    flex: 1;
  }

  .search {
    display: flex;
    align-items: center;
    gap: 6px;
    background: var(--fill-soft);
    border-radius: 8px;
    padding: 4px 9px;
    font-size: var(--text-sm);
    color: var(--fg-muted);
  }

  .search input {
    border: 0;
    background: transparent;
    width: 150px;
    outline: none;
    color: var(--fg);
  }

  .tool {
    display: flex;
    padding: 6px;
    border: 0;
    border-radius: 50%;
    background: none;
    color: var(--fg-muted);
    transition: background var(--dur-fast) var(--ease-out);
  }

  .tool:hover:not(:disabled) {
    background: var(--fill-soft);
  }

  .tool:disabled {
    opacity: 0.5;
  }

  .split {
    flex: 1;
    min-height: 0;
    display: flex;
  }

  .list {
    width: 42%;
    min-width: 220px;
    display: flex;
    flex-direction: column;
    border-right: 1px solid var(--divider-soft);
  }

  .detail {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
  }

  .error {
    display: flex;
    align-items: center;
    gap: 6px;
    margin: 6px 16px 0;
    font-size: var(--text-xs);
    color: var(--warn);
  }

  .footer {
    margin: 0;
    padding: 5px 16px;
    border-top: 1px solid var(--divider-soft);
    font-size: var(--text-xs);
    color: var(--fg-muted);
  }
</style>
