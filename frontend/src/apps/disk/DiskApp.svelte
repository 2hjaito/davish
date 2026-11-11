<script lang="ts">
  import { untrack } from 'svelte'
  import { SvelteSet } from 'svelte/reactivity'

  import type { Cleanup, Mount, Overview, Usage } from '../../lib/api/disk'
  import { toUserError } from '../../lib/api/errors'
  import { withSudo } from '../../lib/stores/sudoPrompt.svelte'
  import { formatBytes } from '../../lib/format'
  import { t } from '../../lib/i18n'
  import { useAppMenu } from '../../lib/shell/menus.svelte'
  import { servers } from '../../lib/stores/servers.svelte'
  import ConfirmCommand from '../../lib/ui/ConfirmCommand.svelte'
  import Empty from '../../lib/ui/Empty.svelte'
  import LoadState from '../../lib/ui/LoadState.svelte'
  import { supersedes } from '../../lib/stores/latest'
  import FilterShell, { type FilterItem } from '../../lib/ui/FilterShell.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import { getWindowContext } from '../../lib/wm/context'
  import { openFilesAt, openTerminalAt, selectRequests } from '../../lib/wm/intents.svelte'
  import type { AppProps } from '../types'
  import { backend } from './backend'
  import OverviewPanel from './OverviewPanel.svelte'
  import RemoveDialog, { type RemoveTarget } from './RemoveDialog.svelte'
  import {
    buildTree,
    matchingRows,
    nameOf,
    parentOf,
    type Row,
    visibleRows,
    withoutPath,
  } from './tree'

  let { win }: AppProps = $props()

  const ctx = getWindowContext()
  const d = t.disk
  const CLEAN = ':clean'

  let filter = $state('/')
  let query = $state('')
  let mounts = $state.raw<Mount[] | null>(null)
  let error = $state('')
  let loading = $state(false)

  // One overview per mount, then the folders measured one level at a time
  // below its depth.
  let overviews = $state.raw<Record<string, Overview>>({})
  let measured = $state.raw<Record<string, Usage>>({})
  const expanded = new SvelteSet<string>()
  let scanning = $state('')
  let measuring = $state('')
  let tab = $state<'dirs' | 'files'>('dirs')
  /** A folder or file path, or a cleanup id. */
  let selected = $state<string | null>(null)

  let cleanups = $state.raw<Cleanup[] | null>(null)
  let confirming = $state<Cleanup | null>(null)
  let removing = $state<RemoveTarget | null>(null)
  let note = $state('')

  const server = $derived(servers.list.find((s) => s.id === win.serverID))
  const connected = $derived(server?.state === 'connected')
  const isClean = $derived(filter === CLEAN)
  const overview = $derived(isClean ? undefined : overviews[filter])

  const pct = (m: Mount) => Math.round((m.used / m.total) * 100)

  const filters = $derived<FilterItem[]>([
    ...(mounts ?? []).map((m) => ({
      id: m.path,
      label: m.path,
      icon: 'disk' as const,
      count: pct(m),
      alert: pct(m) >= 85,
    })),
    { id: CLEAN, label: d.clean, icon: 'trash', count: cleanups?.length },
  ])

  const tree = $derived(buildTree(overview?.dirs ?? [], measured))
  const rows = $derived.by<Row[]>(() => {
    if (!overview) return []
    if (query.trim()) return matchingRows(filter, tree, query)
    return visibleRows(filter, overview.measured, tree, measured, expanded)
  })
  const files = $derived.by(() => {
    const q = query.trim().toLowerCase()
    return (overview?.files ?? []).filter((f) => !q || f.path.toLowerCase().includes(q))
  })
  const shownCleanups = $derived(
    (cleanups ?? []).filter(
      (c) => !query.trim() || d.cleanups[c.id]?.title.toLowerCase().includes(query.trim()),
    ),
  )
  const cleanable = $derived((cleanups ?? []).reduce((s, c) => s + c.bytes, 0))

  const selCleanup = $derived(isClean ? shownCleanups.find((c) => c.id === selected) : undefined)
  const selFile = $derived(
    !isClean && tab === 'files' ? files.find((f) => f.path === selected) : undefined,
  )
  const selDir = $derived(
    !isClean && tab === 'dirs' ? rows.find((r) => r.path === selected) : undefined,
  )
  const footer = $derived(
    isClean ? d.count(shownCleanups.length) : d.count(tab === 'files' ? files.length : rows.length),
  )

  const attempt = supersedes()

  async function refresh() {
    // A retry replaces a load still waiting on the server.
    const current = attempt()
    loading = true
    try {
      mounts = await backend.mounts(win.serverID)
      error = ''
      if (!isClean && !mounts.some((m) => m.path === filter)) filter = mounts[0]?.path ?? '/'
      void loadCleanups()
    } catch (e) {
      if (current()) error = toUserError(e).message
    } finally {
      if (current()) loading = false
    }
  }

  async function loadCleanups() {
    try {
      cleanups = await backend.cleanups(win.serverID)
    } catch (e) {
      cleanups = []
      error = toUserError(e).message
    }
  }

  async function scan(mount: string) {
    if (scanning) return
    const token = `${win.id}-${Date.now()}`
    scanning = token
    error = ''
    try {
      const o = await backend.overview(win.serverID, mount, token)
      overviews = { ...overviews, [mount]: o }
      measured = Object.fromEntries(
        Object.entries(measured).filter(([p]) => !p.startsWith(mount === '/' ? '/' : mount + '/')),
      )
    } catch (e) {
      // A cancel is not an error to show.
      if (scanning === '') return
      error = toUserError(e).message
    } finally {
      if (scanning === token) scanning = ''
    }
  }

  function cancel() {
    const token = scanning || measuring
    scanning = ''
    measuring = ''
    if (token) void backend.cancel(token)
  }

  // Below the overview's depth a folder is measured when opened.
  async function measureDir(path: string) {
    if (measuring) return
    const token = `${win.id}-m-${Date.now()}`
    measuring = token
    try {
      const u = await backend.usage(win.serverID, path, token)
      measured = { ...measured, [path]: u }
      expanded.add(path)
    } catch (e) {
      if (measuring === '') return
      error = toUserError(e).message
    } finally {
      if (measuring === token) measuring = ''
    }
  }

  function toggle(r: Row) {
    if (expanded.has(r.path)) expanded.delete(r.path)
    else if (!r.known) void measureDir(r.path)
    else if (r.hasChildren) expanded.add(r.path)
  }

  async function clean(c: Cleanup) {
    confirming = null
    note = ''
    try {
      const run = () => backend.clean(win.serverID, c.id, c.command)
      if (c.root && c.sudoCommand) {
        // As root through sudo; the prompt shows the command with sudo.
        if (!(await withSudo(win.serverID, c.sudoCommand, run))) return
      } else await run()
      note = d.done
      void refresh()
    } catch (e) {
      note = toUserError(e).message
    }
  }

  function removed(tg: RemoveTarget) {
    removing = null
    selected = null
    note = tg.op === 'remove' ? d.removed(nameOf(tg.path)) : d.truncated(nameOf(tg.path))
    const o = overviews[filter]
    if (o) overviews = { ...overviews, [filter]: withoutPath(o, tg.path, tg.size) }
    measured = Object.fromEntries(
      Object.entries(measured)
        .filter(([p]) => p !== tg.path && !p.startsWith(tg.path + '/'))
        .map(([p, u]) => [p, { ...u, entries: u.entries.filter((e) => e.path !== tg.path) }]),
    )
    void refresh()
  }

  function showFile(path: string) {
    tab = 'files'
    query = ''
    selected = path
  }

  // A new mount starts with its overview; the cleanup list keeps its own.
  $effect(() => {
    const f = filter
    untrack(() => {
      selected = null
      note = ''
      if (f !== CLEAN && connected && mounts && !overviews[f]) void scan(f)
    })
  })

  // The disk notification asked for a mount.
  $effect(() => {
    if (!selectRequests.list.some((r) => r.appId === 'disk' && r.serverID === win.serverID)) return
    untrack(() => {
      const req = selectRequests.take('disk', win.serverID)
      if (req) filter = req.key
    })
  })

  $effect(() => {
    if (connected)
      untrack(async () => {
        await refresh()
        if (!isClean && !overviews[filter]) void scan(filter)
      })
  })
  let wasFocused = untrack(() => ctx.focused)
  $effect(() => {
    const f = ctx.focused
    if (f && !wasFocused && connected) untrack(() => void refresh())
    wasFocused = f
  })

  function onListKey(e: KeyboardEvent) {
    const r = selDir
    if (!r) return
    if (e.key === 'Enter' || (e.key === 'ArrowRight' && !expanded.has(r.path))) toggle(r)
    else if (e.key === 'ArrowLeft' && expanded.has(r.path)) expanded.delete(r.path)
    else if (e.key === 'ArrowLeft' && parentOf(r.path) !== filter) selected = parentOf(r.path)
    else return
    e.preventDefault()
  }

  useAppMenu(
    () => [{ label: d.scan, disabled: !!scanning || isClean, run: () => void scan(filter) }],
    () => void refresh(),
  )
</script>

<FilterShell
  group={server?.name ?? ''}
  {filters}
  bind:filter
  sidebarLabel={d.sections}
  searchLabel={d.search}
  bind:query
  refreshing={!connected || loading}
  ready={connected && mounts !== null}
  {footer}
  {error}
  listWidth="55%"
  onrefresh={() => void refresh()}
>
  {#snippet placeholder()}
    {#if !connected}
      <Empty icon="server" title={d.offline} hint={d.offlineHint} />
    {:else}
      <LoadState
        serverID={win.serverID}
        {error}
        icon="disk"
        loading={d.loading}
        onretry={() => void refresh()}
      />
    {/if}
  {/snippet}
  {#snippet list()}
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="rows" onkeydown={onListKey}>
      {#if isClean}
        {#if cleanups === null}
          <p class="muted pad">{d.cleanLoading}</p>
        {:else if shownCleanups.length === 0}
          <p class="muted pad">{d.cleanEmpty}</p>
        {/if}
        {#each shownCleanups as c (c.id)}
          <button class="row" class:sel={selected === c.id} onclick={() => (selected = c.id)}>
            <span class="name">{d.cleanups[c.id]?.title ?? c.id}</span>
            <span class="size">{formatBytes(c.bytes)}</span>
          </button>
        {/each}
      {:else}
        <div class="crumb">
          <div class="seg" role="group">
            <button class:on={tab === 'dirs'} onclick={() => ((tab = 'dirs'), (selected = null))}
              >{d.tabs.dirs}</button
            >
            <button class:on={tab === 'files'} onclick={() => ((tab = 'files'), (selected = null))}
              >{d.tabs.files}</button
            >
          </div>
          <span class="spacer"></span>
          {#if scanning || measuring}
            <button class="btn small" onclick={cancel}>{d.cancel}</button>
          {:else}
            <button class="btn small" onclick={() => void scan(filter)}>{d.scan}</button>
          {/if}
        </div>
        {#if scanning}
          <p class="muted pad">{d.scanning}</p>
        {:else if !overview}
          <p class="muted pad">{d.loading}</p>
        {:else if tab === 'files'}
          {#if files.length === 0}<p class="muted pad">{d.noFiles}</p>{/if}
          {#each files as f (f.path)}
            <button class="row" class:sel={selected === f.path} onclick={() => (selected = f.path)}>
              <span class="name" title={f.path}
                >{nameOf(f.path)} <span class="dim">{parentOf(f.path)}</span></span
              >
              <span class="size">{formatBytes(f.size)}</span>
            </button>
          {/each}
        {:else}
          {#each rows as r (r.path)}
            {@const open = expanded.has(r.path)}
            <div
              class="row"
              class:sel={selected === r.path}
              style:padding-left="{10 + r.depth * 16}px"
              role="treeitem"
              aria-selected={selected === r.path}
              aria-expanded={r.hasChildren || !r.known ? open : undefined}
              tabindex="0"
              onclick={() => (selected = r.path)}
              ondblclick={() => toggle(r)}
              onkeydown={(e) => e.key === ' ' && (e.preventDefault(), (selected = r.path))}
            >
              {#if !query.trim() && (r.hasChildren || !r.known)}
                <button
                  class="chev"
                  class:open
                  aria-label={open ? d.collapse : d.expand}
                  disabled={!!measuring && !r.known}
                  onclick={(e) => (e.stopPropagation(), toggle(r))}
                >
                  <Icon name="chevronRight" size={12} />
                </button>
              {:else}
                <span class="chev"></span>
              {/if}
              <span class="name" title={r.path}>{r.name}</span>
              {#if measuring && !r.known && !open}<span class="dim">{d.measuring}</span>{/if}
              <span class="bar"
                ><span style:width="{(r.size / Math.max(1, r.parentSize)) * 100}%"></span></span
              >
              <span class="size">{formatBytes(r.size)}</span>
            </div>
          {/each}
        {/if}
      {/if}
    </div>
  {/snippet}
  {#snippet detail()}
    <div class="detail">
      {#if selCleanup}
        {@const c = selCleanup}
        <h3>{d.cleanups[c.id]?.title ?? c.id}</h3>
        <p>
          {d.frees(formatBytes(c.bytes))}{#if c.count}{` · ${d.files(c.count)}`}{/if}
        </p>
        {#if c.danger}<p class="danger">{d.dangerous}</p>{/if}
        <pre class="command">{c.root ? c.sudoCommand : c.command}</pre>
        {#if c.root}<p class="hint">{d.rootRuns}</p>{/if}
        <div class="actions">
          <button class="btn btn-danger" onclick={() => (confirming = c)}>{d.run}</button>
          {#if c.root}
            <button
              class="btn"
              onclick={() => openTerminalAt(win.serverID, undefined, c.sudoCommand)}
              >{d.toTerminal}</button
            >
          {/if}
        </div>
      {:else if selDir && overview}
        {@const r = selDir}
        <h3 class="mono">{r.path}</h3>
        <p>
          {formatBytes(r.size)} · {d.ofDisk(Math.round((r.size / overview.mount.total) * 100))}
          · {d.share(Math.round((r.size / Math.max(1, r.parentSize)) * 100))}
        </p>
        <div class="actions">
          <button class="btn" onclick={() => openFilesAt(win.serverID, r.path)}
            >{d.openFiles}</button
          >
          {#if !r.known || r.path in measured}
            <button class="btn" disabled={!!measuring} onclick={() => void measureDir(r.path)}
              >{d.measureHere}</button
            >
          {/if}
          <button
            class="btn btn-danger"
            onclick={() => (removing = { path: r.path, size: r.size, op: 'remove' })}
            >{d.removeAction}</button
          >
        </div>
      {:else if selFile && overview}
        {@const f = selFile}
        <h3 class="mono">{f.path}</h3>
        <p>{formatBytes(f.size)} · {d.ofDisk(Math.round((f.size / overview.mount.total) * 100))}</p>
        <div class="actions">
          <button class="btn" onclick={() => openFilesAt(win.serverID, parentOf(f.path))}
            >{d.openFolder}</button
          >
          <button
            class="btn"
            onclick={() => (removing = { path: f.path, size: f.size, op: 'truncate' })}
            >{d.truncateAction}</button
          >
          <button
            class="btn btn-danger"
            onclick={() => (removing = { path: f.path, size: f.size, op: 'remove' })}
            >{d.removeAction}</button
          >
        </div>
      {:else if overview}
        <OverviewPanel
          serverID={win.serverID}
          {overview}
          {cleanable}
          onclean={() => (filter = CLEAN)}
          onfile={showFile}
          onrescan={() => {
            void scan(filter)
            void loadCleanups()
          }}
        />
      {:else}
        <p class="muted">{isClean ? d.noSelection : d.scanning}</p>
      {/if}
      {#if note}<p class="note">{note}</p>{/if}
    </div>
  {/snippet}
</FilterShell>

<ConfirmCommand
  command={(confirming?.root ? confirming.sudoCommand : confirming?.command) ?? null}
  title={confirming ? (d.cleanups[confirming.id]?.title ?? '') : ''}
  body={confirming ? (d.cleanups[confirming.id]?.body ?? '') : ''}
  action={d.run}
  onterminal={confirming?.root
    ? () => {
        const c = confirming
        confirming = null
        if (c?.sudoCommand) openTerminalAt(win.serverID, undefined, c.sudoCommand)
      }
    : undefined}
  onclose={() => (confirming = null)}
  onconfirm={() => confirming && void clean(confirming)}
/>

<RemoveDialog
  serverID={win.serverID}
  target={removing}
  onclose={() => (removing = null)}
  ondone={removed}
/>

<style>
  .rows {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    padding: 4px 6px;
  }

  .crumb {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 4px 4px 8px;
  }

  .spacer {
    flex: 1;
  }

  .seg {
    display: flex;
    padding: 2px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
  }

  .seg button {
    padding: 2px 10px;
    border: 0;
    border-radius: calc(var(--control-radius) - 2px);
    background: none;
    font-size: var(--text-sm);
    color: var(--fg-muted);
  }

  .seg button.on {
    background: var(--card-bg);
    color: var(--fg);
    box-shadow: 0 1px 2px rgb(0 0 0 / 12%);
  }

  .chev {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 16px;
    height: 16px;
    flex-shrink: 0;
    padding: 0;
    border: 0;
    background: none;
    color: inherit;
    transition: rotate var(--dur-fast) var(--ease-out);
  }

  .chev.open {
    rotate: 90deg;
  }

  .dim {
    color: var(--fg-subtle);
    font-size: var(--text-xs);
  }

  .row.sel .dim {
    color: inherit;
    opacity: 0.75;
  }

  .small {
    padding: 2px 10px;
    font-size: var(--text-sm);
  }

  .row {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-shrink: 0;
    cursor: default;
    height: 30px;
    padding: 0 10px;
    border: 0;
    border-radius: var(--item-radius);
    background: none;
    text-align: left;
    font-size: var(--text-sm);
  }

  .row.sel {
    background: var(--accent);
    color: var(--fg-on-accent);
  }

  .name {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .bar {
    width: 30%;
    height: 6px;
    border-radius: 3px;
    background: var(--fill-track);
    overflow: hidden;
  }

  .bar span {
    display: block;
    height: 100%;
    background: var(--accent);
  }

  .row.sel .bar span {
    background: var(--fg-on-accent);
  }

  .size {
    width: 72px;
    text-align: right;
    font-variant-numeric: tabular-nums;
  }

  .pad {
    padding: 4px 10px;
  }

  .muted {
    color: var(--fg-muted);
    font-size: var(--text-sm);
  }

  .hint {
    color: var(--fg-muted);
    font-size: var(--text-xs);
  }

  .danger {
    color: var(--danger);
    font-size: var(--text-sm);
  }

  .detail {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
    padding: 16px 18px;
  }

  h3 {
    margin: 0;
    font-size: var(--text-md);
    overflow-wrap: anywhere;
  }

  p {
    margin: 0;
    font-size: var(--text-sm);
  }

  .command {
    align-self: stretch;
    margin: 0;
    padding: 8px 10px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    white-space: pre-wrap;
    overflow-wrap: anywhere;
  }

  .actions {
    display: flex;
    gap: 8px;
  }

  .note {
    color: var(--fg-muted);
  }
</style>
