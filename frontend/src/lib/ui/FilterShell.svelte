<script lang="ts" module>
  import type { IconName } from './icons'

  export interface FilterItem {
    id: string
    label: string
    icon: IconName
    count?: number
    /** Draws the count in red: something needs attention. */
    alert?: boolean
  }
</script>

<script lang="ts">
  import type { Snippet } from 'svelte'

  import { t } from '../i18n/vi'
  import { getWindowContext } from '../wm/context'
  import TitleBar from '../wm/TitleBar.svelte'
  import Icon from './Icon.svelte'

  // The Finder layout of Services and Docker: filters in a translucent
  // sidebar, then a toolbar with search over a list and its detail.
  interface Props {
    group: string
    filters: readonly FilterItem[]
    filter: string
    sidebarLabel: string
    searchLabel: string
    query: string
    refreshing: boolean
    /** False shows placeholder instead of the list and detail. */
    ready: boolean
    /** What to show until ready (offline, loading). */
    placeholder: Snippet
    list: Snippet
    detail: Snippet
    /** Under the list: a count, or an error. */
    footer: string
    error?: string
    onrefresh: () => void
  }

  let {
    group,
    filters,
    filter = $bindable(),
    sidebarLabel,
    searchLabel,
    query = $bindable(),
    refreshing,
    ready,
    placeholder,
    list,
    detail,
    footer,
    error = '',
    onrefresh,
  }: Props = $props()

  const ctx = getWindowContext()
  const title = $derived(filters.find((f) => f.id === filter)?.label ?? '')
</script>

<div class="shell" class:blurred={!ctx.focused}>
  <nav class="sidebar" aria-label={sidebarLabel}>
    <TitleBar bare />
    <div class="items">
      <span class="group">{group}</span>
      {#each filters as f (f.id)}
        <button
          class="item"
          class:active={filter === f.id}
          aria-current={filter === f.id ? 'page' : undefined}
          onclick={() => (filter = f.id)}
        >
          <Icon name={f.icon} size={16} />
          <span class="label">{f.label}</span>
          {#if f.count !== undefined}
            <span class="value" class:alert={f.alert && f.count > 0}>{f.count}</span>
          {/if}
        </button>
      {/each}
    </div>
  </nav>

  <div class="main">
    <!-- Pointer handlers here only add window dragging, as on the title bar. -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="toolbar" onpointerdown={ctx.startDrag} ondblclick={ctx.toggleMaximize}>
      <h2 class="title">{title}</h2>
      <span class="spacer"></span>
      <label class="search" onpointerdown={(e) => e.stopPropagation()}>
        <Icon name="search" size={13} strokeWidth={2.2} />
        <input
          type="text"
          placeholder={searchLabel}
          aria-label={searchLabel}
          bind:value={query}
          onkeydown={(e) => e.key === 'Escape' && (query = '')}
        />
      </label>
      <button
        class="tool"
        aria-label={t.common.refresh}
        title={t.common.refresh}
        disabled={refreshing}
        onpointerdown={(e) => e.stopPropagation()}
        onclick={onrefresh}
      >
        <Icon name="refresh" size={14} strokeWidth={2.2} />
      </button>
    </div>

    {#if !ready}
      {@render placeholder()}
    {:else}
      <div class="split">
        <div class="list">
          {#if error}<p class="error"><Icon name="warning" size={13} /> {error}</p>{/if}
          {@render list()}
          <p class="footer">{footer}</p>
        </div>
        <div class="detail">{@render detail()}</div>
      </div>
    {/if}
  </div>
</div>

<style>
  .shell {
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
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .value {
    font-size: var(--text-xs);
    color: var(--fg-muted);
    font-variant-numeric: tabular-nums;
  }

  .value.alert {
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
    overflow: hidden;
    text-overflow: ellipsis;
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
