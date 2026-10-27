<script lang="ts">
  import { tick, untrack } from 'svelte'

  import { toUserError } from '../../lib/api/errors'
  import { type Entry, filesHome, filesList } from '../../lib/api/files'
  import { hasRuntime } from '../../lib/api/runtime'
  import {
    registerDropTarget,
    transferDownload,
    transferPickFiles,
    transferPickFolder,
    transferPlanUpload,
    transferUpload,
    type UploadItem,
    type UploadRequest,
  } from '../../lib/api/transfer'
  import { formatPct } from '../../lib/format'
  import { t } from '../../lib/i18n/vi'
  import { hasPrimaryModifier, shortcut } from '../../lib/shell/platform'
  import { load, save } from '../../lib/stores/persist'
  import { servers } from '../../lib/stores/servers.svelte'
  import { landingKey, summarize, transfers } from '../../lib/stores/transfers.svelte'
  import Empty from '../../lib/ui/Empty.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import type { IconName } from '../../lib/ui/icons'
  import { getWindowContext } from '../../lib/wm/context'
  import { openFileInEditor, openTransfers } from '../../lib/wm/intents.svelte'
  import TitleBar from '../../lib/wm/TitleBar.svelte'
  import type { AppProps } from '../types'
  import { actionForKey } from './actions'
  import FileActions from './FileActions.svelte'
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
  import { cameFrom, type FolderView, lastDir, rememberDir } from './memory'
  import { mockHome, mockList } from './mockFs'
  import UploadConflictDialog, { type Answer } from './UploadConflictDialog.svelte'

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
  let main = $state<HTMLDivElement>()
  let root = $state<HTMLDivElement>()
  let list = $state<ReturnType<typeof FileList>>()
  let actions = $state<ReturnType<typeof FileActions>>()
  /** A short confirmation in the path bar, like "path copied". */
  let flash = $state('')
  let flashTimer: ReturnType<typeof setTimeout> | undefined
  // Scroll and selection of folders already visited in this window, so
  // coming back lands where the user left off. Not reactive: only read
  // when arriving in a folder.
  // eslint-disable-next-line svelte/prefer-svelte-reactivity -- read on arrival only
  const views = new Map<string, FolderView>()
  /** The name clash being asked about, and how to answer it. */
  let conflict = $state<{
    item: UploadItem
    remaining: number
    answer: (a: Answer) => void
  } | null>(null)

  const server = $derived(servers.list.find((s) => s.id === win.serverID))
  const connected = $derived(!hasRuntime || server?.state === 'connected')
  const entries = $derived(filterEntries(sortEntries(listing, order), query))
  const title = $derived(cwd === null ? (server?.name ?? '') : baseName(cwd))
  const busy = $derived(summarize(transfers.list, win.serverID))
  const canTransfer = $derived(hasRuntime && connected && cwd !== null)

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

  /**
   * Lists path and shows it; false when it could not be listed. quiet keeps
   * a failure off screen, for guesses like the last folder of a past run.
   */
  async function show(path: string, quiet = false): Promise<boolean> {
    const id = win.serverID
    const mine = ++seq
    loading = true
    try {
      const entries = hasRuntime ? await filesList(id, path) : mockList(path)
      if (mine !== seq) return false
      const from = cwd
      const arriving = path !== from
      if (arriving && from !== null) {
        views.set(from, { scrollTop: list?.getScroll() ?? 0, selected })
      }
      const view = arriving ? views.get(path) : undefined
      if (arriving) {
        query = ''
        selected = (from && cameFrom(from, path)) ?? view?.selected ?? null
      }
      if (selected !== null && !entries.some((e) => e.path === selected)) selected = null
      cwd = path
      listing = entries
      fetchedAt = new Date()
      error = ''
      rememberDir(id, path)
      if (arriving) {
        await tick()
        list?.restore(view?.scrollTop ?? 0, selected)
      }
      return true
    } catch (e) {
      if (mine === seq && !quiet) error = toUserError(e).message
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
    // Back where the last run left this server, if that folder still opens.
    const last = lastDir(win.serverID)
    if (last && last !== home && (await show(last, true))) return
    await show(home)
  }

  function reload() {
    if (cwd === null) void start()
    else void show(cwd)
  }

  /** After a change on the server: list again, then select path if given. */
  async function changed(path: string | null) {
    if (cwd === null || !(await show(cwd))) return
    if (path === null || !listing.some((e) => e.path === path)) return
    selected = path
    await tick()
    list?.restore(null, path)
  }

  function showFlash(text: string) {
    flash = text
    clearTimeout(flashTimer)
    flashTimer = setTimeout(() => (flash = ''), 2500)
  }

  /** Shortcuts for the drag menu's actions; false when the key is not one. */
  function actionKey(e: KeyboardEvent): boolean {
    // Typing in the search box or a dialog never touches files.
    if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement)
      return false
    const id = actionForKey({
      key: e.key,
      code: e.code,
      primary: hasPrimaryModifier(e),
      shift: e.shiftKey,
      alt: e.altKey,
    })
    if (!id || !connected) return false
    const entry = listing.find((x) => x.path === selected) ?? null
    if (!entry && id !== 'terminal') return false
    e.preventDefault()
    actions?.act(id, entry)
    return true
  }

  // Load on open and again whenever the server comes back.
  $effect(() => {
    if (connected) untrack(reload)
  })

  // Uploads that finish in the folder on screen show up without a refresh.
  let seen: number | undefined
  $effect(() => {
    const n = cwd === null ? undefined : transfers.landed[landingKey(win.serverID, cwd)]
    if (n !== undefined && seen !== undefined && n !== seen) untrack(reload)
    seen = n
  })

  // Files dropped from the OS upload into the folder on screen.
  $effect(() => {
    if (main) return registerDropTarget(main, (paths) => void upload(paths))
  })

  function ask(item: UploadItem, remaining: number): Promise<Answer> {
    return new Promise((resolve) => {
      conflict = {
        item,
        remaining,
        answer: (a) => {
          conflict = null
          resolve(a)
        },
      }
    })
  }

  /** Uploads local paths into the folder on screen, asking about name clashes one by one. */
  async function upload(locals: string[]) {
    const dir = cwd
    if (!canTransfer || dir === null || locals.length === 0) return
    const id = win.serverID
    try {
      const items = await transferPlanUpload(id, dir, locals)
      const reqs: UploadRequest[] = []
      let clashes = items.filter((it) => it.exists).length
      for (const it of items) {
        if (!it.exists) {
          reqs.push({ local: it.local, replace: false })
          continue
        }
        const a = await ask(it, --clashes)
        if (a !== 'skip') reqs.push({ local: it.local, replace: a === 'replace' })
      }
      if (reqs.length > 0) transfers.add(await transferUpload(id, dir, reqs))
      error = ''
    } catch (e) {
      error = toUserError(e).message
    }
  }

  async function pickAndUpload(folder: boolean) {
    if (!canTransfer) return
    try {
      const picked = folder
        ? [await transferPickFolder(t.files.pickFolder)].filter(Boolean)
        : await transferPickFiles(t.files.pickFiles)
      await upload(picked)
    } catch (e) {
      error = toUserError(e).message
    }
  }

  async function download() {
    if (!canTransfer || selected === null) return
    try {
      transfers.add(await transferDownload(win.serverID, [selected]))
      error = ''
    } catch (e) {
      error = toUserError(e).message
    }
  }

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
    if (e.dir) void go(e.path)
    else openFileInEditor(win.serverID, e.path)
  }

  function sortBy(key: SortKey) {
    order = nextOrder(order, key)
  }

  function onkeydown(e: KeyboardEvent) {
    if (actionKey(e)) return
    // ⌘⌥L like Safari's downloads; code, since Option changes e.key on macOS.
    if (hasPrimaryModifier(e) && e.altKey && !e.shiftKey && e.code === 'KeyL') {
      e.preventDefault()
      openTransfers(win.serverID)
      return
    }
    if (!hasPrimaryModifier(e) || e.altKey) return
    const actions: Record<string, () => void> = e.shiftKey
      ? { u: () => void pickAndUpload(true) }
      : {
          ArrowUp: () => cwd !== null && void go(parentPath(cwd)),
          '[': () => void goBack(),
          ']': () => void goForward(),
          f: () => search?.focus(),
          u: () => void pickAndUpload(false),
          d: () => void download(),
        }
    const action = actions[e.key.toLowerCase()] ?? actions[e.key]
    if (!action) return
    e.preventDefault()
    action()
  }
</script>

<!-- Shortcuts for the whole window; every one also has a button. -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="files" {onkeydown} bind:this={root}>
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

  <div class="main" bind:this={main}>
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
        aria-label={t.files.upload}
        title="{t.files.upload} ({shortcut('U')}) · {t.files.uploadFolder} ({shortcut('⇧U')})"
        disabled={!canTransfer}
        onclick={(e) => void pickAndUpload(e.altKey)}
      >
        <Icon name="upload" size={15} strokeWidth={2.2} />
      </button>
      <button
        class="tool"
        aria-label={t.files.download}
        title="{t.files.download} ({shortcut('D')})"
        disabled={!canTransfer || selected === null}
        onclick={() => void download()}
      >
        <Icon name="download" size={15} strokeWidth={2.2} />
      </button>
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
        ondrag={(entry, phase, x, y) => actions?.drag(entry, phase, x, y)}
        bind:this={list}
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
      {#if flash}
        <span class="count flash" role="status">{flash}</span>
      {:else if busy.count > 0}
        <button
          class="count busy"
          title="{t.transfers.open} ({shortcut('⌥L')})"
          onclick={() => openTransfers(win.serverID)}
          >{t.files.transferring(busy.count, formatPct(busy.fraction * 100))}</button
        >
      {:else if cwd !== null && connected}
        <span class="count">{t.files.count(listing.length)}</span>
      {/if}
    </nav>
  </div>
</div>

<FileActions
  bind:this={actions}
  serverID={win.serverID}
  {cwd}
  bounds={() => root?.getBoundingClientRect()}
  onchanged={(path) => void changed(path)}
  onflash={showFlash}
/>

<UploadConflictDialog
  item={conflict?.item ?? null}
  remaining={conflict?.remaining ?? 0}
  onanswer={(a) => conflict?.answer(a)}
/>

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

  /* Wails delivers OS file drops only over elements with this property. */
  .main {
    --wails-drop-target: drop;
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    background: var(--content-bg);
    border-left: 1px solid var(--divider-soft);
  }

  /* Wails adds this class while files are dragged over the window. */
  .main:global(.wails-drop-target-active) {
    outline: 2px solid var(--accent);
    outline-offset: -2px;
    background: var(--accent-soft);
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

  .count.flash {
    color: var(--ok);
  }

  .count.busy {
    border: 0;
    border-radius: 4px;
    padding: 1px 6px;
    background: none;
    color: var(--accent);
    font-size: inherit;
  }

  .count.busy:hover {
    background: var(--accent-soft);
  }
</style>
