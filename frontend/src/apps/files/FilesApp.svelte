<script lang="ts">
  import { untrack } from 'svelte'

  import { toUserError } from '../../lib/api/errors'
  import { type Entry, filesHome, filesList } from '../../lib/api/files'
  import { hasRuntime } from '../../lib/api/runtime'
  import { t } from '../../lib/i18n/vi'
  import { hasPrimaryModifier, shortcut } from '../../lib/shell/platform'
  import { load, save } from '../../lib/stores/persist'
  import { servers } from '../../lib/stores/servers.svelte'
  import Empty from '../../lib/ui/Empty.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import type { IconName } from '../../lib/ui/icons'
  import { getWindowContext } from '../../lib/wm/context'
  import TitleBar from '../../lib/wm/TitleBar.svelte'
  import type { AppProps } from '../types'
  import FileList from './FileList.svelte'
  import {
    baseName,
    crumbs,
    filterEntries,
    nextOrder,
    parentPath,
    SORT_KEYS,
    type SortKey,
    type SortOrder,
    sortEntries,
    within,
  } from './listing'
  import { mockHome, mockList } from './mockFs'

  let { win }: AppProps = $props()

  const ctx = getWindowContext()

  const saved = load('files', { order: { key: 'name', desc: false } as SortOrder })
  let order = $state<SortOrder>(
    SORT_KEYS.includes(saved.order.key) ? saved.order : { key: 'name', desc: false },
  )
  $effect(() => save('files', { order }))

  let home = $state<string | null>(null)
  /** The folder on screen; it only changes once its listing has arrived. */
  let cwd = $state<string | null>(null)
  let listing = $state.raw<Entry[]>([])
  let fetchedAt = $state(new Date())
  let loading = $state(false)
  let error = $state('')
  let query = $state('')
  let selected = $state<string | null>(null)
  let back = $state<string[]>([])
  let forward = $state<string[]>([])
  let search = $state<HTMLInputElement>()

  const server = $derived(servers.list.find((s) => s.id === win.serverID))
  const connected = $derived(!hasRuntime || server?.state === 'connected')
  const entries = $derived(filterEntries(sortEntries(listing, order), query))
  const title = $derived(cwd === null ? (server?.name ?? '') : baseName(cwd))

  interface Place {
    label: string
    path: string
    icon: IconName
  }

  const places = $derived<Place[]>([
    ...(home ? [{ label: t.files.home, path: home, icon: 'home' as const }] : []),
    { label: t.files.root, path: '/', icon: 'disk' },
    ...['/etc', '/var/log', '/opt', '/tmp'].map((path) => ({
      label: path,
      path,
      icon: 'folder' as const,
    })),
  ])
  // The most specific place holding cwd, so /etc/nginx lights up /etc, not /.
  const activePlace = $derived(
    cwd === null
      ? null
      : places
          .filter((p) => within(cwd ?? '', p.path))
          .reduce<Place | null>((a, p) => (a && a.path.length >= p.path.length ? a : p), null),
  )

  // Requests can overlap (double clicks, slow links); only the latest counts.
  let seq = 0

  /** Lists path and shows it; false when it could not be listed. */
  async function show(path: string): Promise<boolean> {
    const id = win.serverID
    const mine = ++seq
    loading = true
    try {
      const list = hasRuntime ? await filesList(id, path) : mockList(path)
      if (mine !== seq) return false
      if (path !== cwd) {
        query = ''
        selected = null
      } else if (selected !== null && !list.some((e) => e.path === selected)) {
        selected = null
      }
      cwd = path
      listing = list
      fetchedAt = new Date()
      error = ''
      return true
    } catch (e) {
      if (mine === seq) error = toUserError(e).message
      return false
    } finally {
      if (mine === seq) loading = false
    }
  }

  async function start() {
    try {
      home = hasRuntime ? await filesHome(win.serverID) : mockHome
    } catch (e) {
      error = toUserError(e).message
      return
    }
    await show(home)
  }

  function reload() {
    if (cwd === null) void start()
    else void show(cwd)
  }

  // Load on open and again whenever the server comes back.
  $effect(() => {
    if (connected) untrack(reload)
  })

  async function go(path: string) {
    if (path === cwd) return
    const from = cwd
    if ((await show(path)) && from !== null) {
      back.push(from)
      forward = []
    }
  }

  async function goBack() {
    const path = back.at(-1)
    if (path === undefined || cwd === null) return
    const from = cwd
    if (await show(path)) {
      back.pop()
      forward.push(from)
    }
  }

  async function goForward() {
    const path = forward.at(-1)
    if (path === undefined || cwd === null) return
    const from = cwd
    if (await show(path)) {
      forward.pop()
      back.push(from)
    }
  }

  function openEntry(e: Entry) {
    // Files open in the Editor from M6 on.
    if (e.dir) void go(e.path)
  }

  function sortBy(key: SortKey) {
    order = nextOrder(order, key)
  }

  function onkeydown(e: KeyboardEvent) {
    if (!hasPrimaryModifier(e) || e.shiftKey || e.altKey) return
    const actions: Record<string, () => void> = {
      ArrowUp: () => cwd !== null && void go(parentPath(cwd)),
      '[': () => void goBack(),
      ']': () => void goForward(),
      f: () => search?.focus(),
    }
    const action = actions[e.key]
    if (!action) return
    e.preventDefault()
    action()
  }
</script>

<!-- Shortcuts for the whole window; every one also has a button. -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="files" {onkeydown}>
  <nav class="sidebar" aria-label={t.files.places}>
    <TitleBar bare />
    <div class="places">
      <span class="group">{server?.name ?? ''}</span>
      {#each places as p (p.path)}
        <button
          class="place"
          class:active={activePlace?.path === p.path}
          disabled={!connected}
          onclick={() => void go(p.path)}
        >
          <Icon name={p.icon} size={16} />
          <span class="label">{p.label}</span>
        </button>
      {/each}
    </div>
  </nav>

  <div class="main">
    <!-- Pointer handlers here only add window dragging, as on the title bar. -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="toolbar" onpointerdown={ctx.startDrag} ondblclick={ctx.toggleMaximize}>
      <div class="nav">
        <button
          aria-label={t.files.back}
          title="{t.files.back} ({shortcut('[')})"
          disabled={back.length === 0}
          onclick={() => void goBack()}
        >
          <Icon name="chevronLeft" size={18} strokeWidth={2.2} />
        </button>
        <button
          aria-label={t.files.forward}
          title="{t.files.forward} ({shortcut(']')})"
          disabled={forward.length === 0}
          onclick={() => void goForward()}
        >
          <Icon name="chevronRight" size={18} strokeWidth={2.2} />
        </button>
        <button
          aria-label={t.files.up}
          title="{t.files.up} ({shortcut('↑')})"
          disabled={cwd === null || cwd === '/'}
          onclick={() => cwd !== null && void go(parentPath(cwd))}
        >
          <Icon name="arrowUp" size={16} strokeWidth={2.2} />
        </button>
      </div>
      <h2 class="title" title={cwd ?? ''}>{title}</h2>
      <button
        class="tool"
        aria-label={t.files.refresh}
        title={t.files.refresh}
        disabled={!connected || loading}
        onclick={reload}
      >
        <Icon name="refresh" size={14} strokeWidth={2.2} />
      </button>
      <label class="search">
        <Icon name="search" size={13} strokeWidth={2.2} />
        <input
          type="text"
          placeholder={t.files.search}
          aria-label={t.files.searchLabel}
          bind:value={query}
          bind:this={search}
          onkeydown={(e) => e.key === 'Escape' && (query = '')}
        />
      </label>
    </div>

    {#if error}
      <div class="error" role="alert">
        <Icon name="warning" size={14} strokeWidth={2.2} />
        <span>{error}</span>
        <button onclick={reload}>{t.files.retry}</button>
      </div>
    {/if}

    {#if !connected}
      <Empty icon="server" title={t.files.offline} hint={t.files.offlineHint} />
    {:else if cwd === null}
      {#if !error}<Empty icon="folder" title={t.files.loading} />{/if}
    {:else if entries.length === 0}
      <Empty icon="folder" title={query ? t.files.noMatch : t.files.empty} />
    {:else}
      <FileList
        {entries}
        {order}
        {selected}
        now={fetchedAt}
        onsort={sortBy}
        onselect={(path) => (selected = path)}
        onopen={openEntry}
      />
    {/if}

    <nav class="pathbar" aria-label={t.files.path}>
      <button class="crumb" disabled={!connected} onclick={() => void go('/')}>
        <Icon name="server" size={12} />
        {server?.name ?? '/'}
      </button>
      {#each crumbs(cwd ?? '/') as c (c.path)}
        <Icon name="chevronRight" size={10} strokeWidth={2.4} />
        <button class="crumb" onclick={() => void go(c.path)}>
          <Icon name="folder" size={12} />
          {c.name}
        </button>
      {/each}
      {#if cwd !== null && connected}
        <span class="count">{t.files.count(listing.length)}</span>
      {/if}
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

  .place .label {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
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

  .nav button,
  .tool {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 26px;
    height: 26px;
    padding: 0;
    border: 0;
    border-radius: var(--control-radius);
    background: none;
    color: var(--fg-muted);
    transition: background var(--dur-fast) var(--ease-out);
  }

  .nav button:hover:not(:disabled),
  .tool:hover:not(:disabled) {
    background: var(--fill-soft);
  }

  .nav button:disabled,
  .tool:disabled {
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

  .error {
    display: flex;
    align-items: center;
    gap: 8px;
    margin: 8px 8px 0;
    padding: 6px 10px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    font-size: var(--text-sm);
    color: var(--fg);
  }

  .error :global(svg) {
    color: var(--warn);
    flex-shrink: 0;
  }

  .error span {
    flex: 1;
    min-width: 0;
  }

  .error button {
    border: 0;
    background: none;
    padding: 2px 6px;
    border-radius: var(--control-radius);
    color: var(--accent);
    font-weight: 600;
  }

  .error button:hover {
    background: var(--accent-soft);
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

  .crumb:hover:not(:disabled) {
    background: var(--fill-soft);
  }

  .count {
    margin-left: auto;
    padding-left: 12px;
    font-variant-numeric: tabular-nums;
  }
</style>
