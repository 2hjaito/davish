<script lang="ts">
  import { formatBytes } from '../../lib/format'
  import { t } from '../../lib/i18n/vi'
  import { servers } from '../../lib/stores/servers.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import { getWindowContext } from '../../lib/wm/context'
  import TitleBar from '../../lib/wm/TitleBar.svelte'
  import type { AppProps } from '../types'
  import { join, mockFs, places, remotePlaces } from './mockFs'

  let { win }: AppProps = $props()

  const ctx = getWindowContext()

  let cwd = $state('/etc/nginx')
  let query = $state('')
  let selected = $state<string | null>(null)
  let back = $state<string[]>([])
  let forward = $state<string[]>([])

  const serverName = $derived(servers.list.find((s) => s.id === win.serverID)?.name ?? '')
  const otherServers = $derived(servers.list.filter((s) => s.id !== win.serverID))
  const crumbs = $derived(
    cwd
      .split('/')
      .filter(Boolean)
      .map((name, i, all) => ({ name, path: '/' + all.slice(0, i + 1).join('/') })),
  )
  const folderName = $derived(crumbs.at(-1)?.name ?? serverName)
  const entries = $derived(
    (mockFs[cwd] ?? []).filter((e) => e.name.toLowerCase().includes(query.trim().toLowerCase())),
  )

  function show(path: string) {
    cwd = path
    query = ''
    selected = null
  }

  function go(path: string) {
    if (path === cwd) return
    back.push(cwd)
    forward = []
    show(path)
  }

  function goBack() {
    const path = back.pop()
    if (path === undefined) return
    forward.push(cwd)
    show(path)
  }

  function goForward() {
    const path = forward.pop()
    if (path === undefined) return
    back.push(cwd)
    show(path)
  }

  function isActivePlace(path: string) {
    return cwd === path || cwd.startsWith(path + '/')
  }
</script>

<div class="files" class:blurred={!ctx.focused}>
  <nav class="sidebar" aria-label={t.files.places}>
    <TitleBar bare />
    <div class="places">
      <span class="group">{serverName}</span>
      {#each places as p (p.path)}
        <button class="place" class:active={isActivePlace(p.path)} onclick={() => go(p.path)}>
          <Icon name={p.icon} size={16} />
          {p.label}
        </button>
      {/each}
      {#each otherServers as s (s.id)}
        <span class="group">{s.name}</span>
        {#each remotePlaces as p (p.path)}
          <button class="place" disabled>
            <Icon name={p.icon} size={16} />
            {p.label}
          </button>
        {/each}
      {/each}
    </div>
  </nav>

  <div class="main">
    <!-- Pointer handlers here only add window dragging, as on the title bar. -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="toolbar" onpointerdown={ctx.startDrag} ondblclick={ctx.toggleMaximize}>
      <div class="nav">
        <button aria-label={t.files.back} disabled={back.length === 0} onclick={goBack}>
          <Icon name="chevronLeft" size={18} strokeWidth={2.2} />
        </button>
        <button aria-label={t.files.forward} disabled={forward.length === 0} onclick={goForward}>
          <Icon name="chevronRight" size={18} strokeWidth={2.2} />
        </button>
      </div>
      <h2 class="title">{folderName}</h2>
      <label class="search">
        <Icon name="search" size={13} strokeWidth={2.2} />
        <input
          type="text"
          placeholder={t.files.search}
          aria-label={t.files.searchLabel}
          bind:value={query}
        />
      </label>
    </div>

    <div class="list">
      <div class="row head">
        <span>{t.files.name}</span><span class="size">{t.files.size}</span><span
          >{t.files.modified}</span
        >
      </div>
      {#each entries as e (e.name)}
        <button
          class="row"
          class:selected={selected === e.name}
          onclick={() => (selected = e.name)}
          ondblclick={() => e.dir && go(join(cwd, e.name))}
        >
          <span class="name" class:dir={e.dir}>
            <Icon name={e.dir ? 'folder' : 'file'} size={16} strokeWidth={e.dir ? 1.5 : 1.8} />
            <span class="label">{e.name}</span>
          </span>
          <span class="meta size">
            {#if e.size === undefined}<span class="none">--</span>{:else}{formatBytes(e.size)}{/if}
          </span>
          <span class="meta">{e.modified}</span>
        </button>
      {:else}
        <p class="empty">{query ? t.files.noMatch : t.files.empty}</p>
      {/each}
    </div>

    <nav class="pathbar" aria-label={t.files.path}>
      <button class="crumb" onclick={() => go('/')}>
        <Icon name="server" size={12} />
        {serverName}
      </button>
      {#each crumbs as c (c.path)}
        <Icon name="chevronRight" size={10} strokeWidth={2.4} />
        <button class="crumb" onclick={() => go(c.path)}>
          <Icon name="folder" size={12} />
          {c.name}
        </button>
      {/each}
    </nav>
  </div>
</div>

<style>
  /* Finder layout: a translucent sidebar running the full height, beside an
     almost opaque content pane so file names read cleanly over any wallpaper. */
  .files {
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

  .places {
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
  }

  .group:not(:first-child) {
    padding-top: 14px;
  }

  .place {
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

  .place :global(svg) {
    color: var(--accent);
    flex-shrink: 0;
  }

  .place:hover:not(:disabled, .active) {
    background: var(--row-hover);
  }

  .place.active {
    background: var(--sidebar-active);
  }

  .place:disabled {
    color: var(--fg-muted);
  }

  .place:disabled :global(svg) {
    color: var(--fg-subtle);
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
    padding: 0 10px 0 8px;
    border-bottom: 1px solid var(--divider-soft);
  }

  .nav {
    display: flex;
  }

  .nav button {
    display: flex;
    padding: 3px;
    border: 0;
    border-radius: var(--control-radius);
    background: none;
    color: var(--fg-muted);
    transition: background var(--dur-fast) var(--ease-out);
  }

  .nav button:hover:not(:disabled) {
    background: var(--fill-soft);
  }

  .nav button:disabled {
    color: var(--fg-subtle);
    opacity: 0.5;
  }

  .title {
    flex: 1;
    min-width: 0;
    margin: 0;
    font-size: var(--text-lg);
    font-weight: 600;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .search {
    display: flex;
    align-items: center;
    gap: 6px;
    background: var(--fill-soft);
    border-radius: 8px;
    padding: 5px 9px;
    font-size: var(--text-sm);
    color: var(--fg-muted);
  }

  .search input {
    border: 0;
    background: transparent;
    width: 110px;
    outline: none;
    color: var(--fg);
  }

  .list {
    flex: 1;
    min-height: 0;
    padding: 0 8px 6px;
    font-size: var(--text-md);
    overflow-y: auto;
  }

  .row {
    width: 100%;
    display: grid;
    grid-template-columns: minmax(0, 1fr) 76px 100px;
    gap: 14px;
    padding: 6px 10px;
    border: 0;
    border-radius: var(--control-radius);
    background: transparent;
    align-items: center;
    text-align: left;
  }

  /* Zebra stripes, like Finder's list view, guide the eye along a row. */
  button.row:nth-of-type(even) {
    background: var(--row-stripe);
  }

  button.row:hover {
    background: var(--row-hover);
  }

  /* Like Finder: a solid accent row, clearly apart from the hover tint; it
     turns gray while the window is in the background. */
  button.row.selected {
    background: var(--accent);
  }

  .row.head {
    position: sticky;
    top: 0;
    z-index: 1;
    margin: 0 -8px 4px;
    width: calc(100% + 16px);
    padding: 6px 18px;
    border-radius: 0;
    background: var(--content-bg);
    border-bottom: 1px solid var(--divider-soft);
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--fg-muted);
  }

  .name {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
    color: var(--file-stroke);
  }

  .name.dir {
    color: var(--folder-stroke);
  }

  .name.dir :global(svg *) {
    fill: var(--folder-fill);
  }

  .label {
    color: var(--fg);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .meta {
    color: var(--fg-muted);
  }

  /* Like Finder: sizes line up on the right so their magnitudes compare at a glance. */
  .size {
    text-align: right;
    font-variant-numeric: tabular-nums;
  }

  /* Folders have no size. Plain figures keep the two dashes tight. */
  .none {
    font-variant-numeric: normal;
    color: var(--fg-subtle);
  }

  .selected :is(.name, .label, .meta) {
    color: var(--fg-on-accent);
  }

  .selected .name.dir :global(svg *) {
    fill: var(--fg-on-accent);
  }

  .blurred button.row.selected {
    background: var(--selection-inactive);
  }

  .blurred .selected :is(.name, .label, .meta) {
    color: var(--fg);
  }

  .blurred .selected .name.dir :global(svg *) {
    fill: var(--folder-fill);
  }

  .empty {
    padding: 20px 10px;
    color: var(--fg-subtle);
    text-align: center;
  }

  .pathbar {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    gap: 2px;
    padding: 4px 10px;
    border-top: 1px solid var(--divider-soft);
    font-size: var(--text-xs);
    color: var(--fg-subtle);
    white-space: nowrap;
    overflow: hidden;
  }

  .crumb {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 1px 4px;
    border: 0;
    border-radius: 4px;
    background: none;
    color: var(--fg-muted);
  }

  .crumb :global(svg) {
    color: var(--accent);
  }

  .crumb:hover {
    background: var(--fill-soft);
  }
</style>
