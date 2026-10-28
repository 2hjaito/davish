<script lang="ts">
  import { untrack } from 'svelte'

  import type { Cleanup, Mount, Usage } from '../../lib/api/disk'
  import { toUserError } from '../../lib/api/errors'
  import { formatBytes } from '../../lib/format'
  import { t } from '../../lib/i18n/vi'
  import { useAppMenu } from '../../lib/shell/menus.svelte'
  import { servers } from '../../lib/stores/servers.svelte'
  import ConfirmCommand from '../../lib/ui/ConfirmCommand.svelte'
  import Empty from '../../lib/ui/Empty.svelte'
  import FilterShell, { type FilterItem } from '../../lib/ui/FilterShell.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import { getWindowContext } from '../../lib/wm/context'
  import { openFilesAt, openTerminalAt, selectRequests } from '../../lib/wm/intents.svelte'
  import type { AppProps } from '../types'
  import { backend } from './backend'

  let { win }: AppProps = $props()

  const ctx = getWindowContext()
  const CLEAN = ':clean'

  let filter = $state('/')
  let query = $state('')
  let mounts = $state.raw<Mount[] | null>(null)
  let error = $state('')
  let loading = $state(false)

  // Measurements by folder, so going back up shows what was measured.
  let usages = $state.raw<Record<string, Usage>>({})
  let dir = $state('/')
  let scanning = $state('')
  let selected = $state<string | null>(null)

  let cleanups = $state.raw<Cleanup[] | null>(null)
  let confirming = $state<Cleanup | null>(null)
  let note = $state('')

  const server = $derived(servers.list.find((s) => s.id === win.serverID))
  const connected = $derived(server?.state === 'connected')
  const usage = $derived(usages[dir])
  const isClean = $derived(filter === CLEAN)

  const pct = (m: Mount) => Math.round((m.used / m.total) * 100)
  const base = (p: string) => p.slice(p.lastIndexOf('/') + 1) || p

  const filters = $derived<FilterItem[]>([
    ...(mounts ?? []).map((m) => ({
      id: m.path,
      label: m.path,
      icon: 'disk' as const,
      count: pct(m),
      alert: pct(m) >= 85,
    })),
    { id: CLEAN, label: t.disk.clean, icon: 'trash', count: cleanups?.length },
  ])

  const entries = $derived.by(() => {
    const q = query.trim().toLowerCase()
    return (usage?.entries ?? []).filter((e) => !q || base(e.path).toLowerCase().includes(q))
  })
  const shownCleanups = $derived(
    (cleanups ?? []).filter(
      (c) => !query.trim() || t.disk.cleanups[c.id]?.title.toLowerCase().includes(query.trim()),
    ),
  )
  const largest = $derived(Math.max(1, ...entries.map((e) => e.size)))
  const current = $derived(
    isClean
      ? shownCleanups.find((c) => c.id === selected)
      : entries.find((e) => e.path === selected),
  )

  async function refresh() {
    if (loading) return
    loading = true
    try {
      mounts = await backend.mounts(win.serverID)
      error = ''
      if (!isClean && !mounts.some((m) => m.path === filter)) filter = mounts[0]?.path ?? '/'
      void loadCleanups()
    } catch (e) {
      error = toUserError(e).message
    } finally {
      loading = false
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

  async function measure(path: string) {
    if (scanning) return
    const token = `${win.id}-${Date.now()}`
    scanning = token
    dir = path
    selected = null
    error = ''
    try {
      const u = await backend.usage(win.serverID, path, token)
      usages = { ...usages, [path]: u }
    } catch (e) {
      // A cancel is not an error to show.
      if (scanning === '') return
      error = toUserError(e).message
    } finally {
      if (scanning === token) scanning = ''
    }
  }

  function cancel() {
    const token = scanning
    scanning = ''
    void backend.cancel(token)
  }

  function up() {
    if (dir === filter) return
    const parent = dir.slice(0, dir.lastIndexOf('/')) || '/'
    dir = parent.length < filter.length ? filter : parent
    selected = null
  }

  function open(path: string) {
    if (usages[path]) {
      dir = path
      selected = null
    } else void measure(path)
  }

  async function clean(c: Cleanup) {
    confirming = null
    note = ''
    try {
      await backend.clean(win.serverID, c.id, c.command)
      note = t.disk.done
      void refresh()
    } catch (e) {
      note = toUserError(e).message
    }
  }

  // A new mount starts at its root; the cleanup list keeps its own state.
  $effect(() => {
    const f = filter
    untrack(() => {
      selected = null
      note = ''
      if (f !== CLEAN) dir = f
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
    if (connected) untrack(() => void refresh())
  })
  let wasFocused = untrack(() => ctx.focused)
  $effect(() => {
    const f = ctx.focused
    if (f && !wasFocused && connected) untrack(() => void refresh())
    wasFocused = f
  })

  function onListKey(e: KeyboardEvent) {
    if (isClean) return
    if (e.key === 'Enter' && selected) open(selected)
    else if (e.key === 'Backspace') up()
    else return
    e.preventDefault()
  }

  useAppMenu(
    () => [
      { label: t.disk.measure, disabled: !!scanning || isClean, run: () => void measure(dir) },
    ],
    () => void refresh(),
  )
</script>

<FilterShell
  group={server?.name ?? ''}
  {filters}
  bind:filter
  sidebarLabel={t.disk.sections}
  searchLabel={t.disk.search}
  bind:query
  refreshing={!connected || loading}
  ready={connected && mounts !== null}
  footer={isClean ? t.disk.count(shownCleanups.length) : t.disk.count(entries.length)}
  {error}
  listWidth="55%"
  onrefresh={() => void refresh()}
>
  {#snippet placeholder()}
    {#if !connected}
      <Empty icon="server" title={t.disk.offline} hint={t.disk.offlineHint} />
    {:else}
      <Empty icon={error ? 'warning' : 'disk'} title={error || t.disk.loading} />
    {/if}
  {/snippet}
  {#snippet list()}
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="rows" onkeydown={onListKey}>
      {#if isClean}
        {#if cleanups === null}
          <p class="muted pad">{t.disk.cleanLoading}</p>
        {:else if shownCleanups.length === 0}
          <p class="muted pad">{t.disk.cleanEmpty}</p>
        {/if}
        {#each shownCleanups as c (c.id)}
          <button class="row" class:sel={selected === c.id} onclick={() => (selected = c.id)}>
            <span class="name">{t.disk.cleanups[c.id]?.title ?? c.id}</span>
            <span class="size">{formatBytes(c.bytes)}</span>
          </button>
        {/each}
      {:else}
        <div class="crumb">
          <button class="tool" aria-label={t.disk.up} disabled={dir === filter} onclick={up}>
            <Icon name="arrowRight" size={13} />
          </button>
          <span class="path mono" title={dir}>{dir}</span>
          {#if scanning}
            <button class="btn small" onclick={cancel}>{t.disk.cancel}</button>
          {:else}
            <button class="btn small" onclick={() => void measure(dir)}>{t.disk.measure}</button>
          {/if}
        </div>
        {#if scanning}
          <p class="muted pad">{t.disk.measuring}</p>
        {:else if !usage}
          <p class="muted pad">{t.disk.empty}</p>
          <p class="hint pad">{t.disk.measureHint}</p>
        {:else}
          {#if usage.partial}<p class="warn pad">{t.disk.partial}</p>{/if}
          {#if entries.length === 0}<p class="muted pad">{t.disk.noChildren}</p>{/if}
          {#each entries as e (e.path)}
            <button
              class="row"
              class:sel={selected === e.path}
              onclick={() => (selected = e.path)}
              ondblclick={() => open(e.path)}
            >
              <span class="name">{base(e.path)}</span>
              <span class="bar"><span style:width="{(e.size / largest) * 100}%"></span></span>
              <span class="size">{formatBytes(e.size)}</span>
            </button>
          {/each}
        {/if}
      {/if}
    </div>
  {/snippet}
  {#snippet detail()}
    <div class="detail">
      {#if isClean && current && 'command' in current}
        {@const c = current}
        <h3>{t.disk.cleanups[c.id]?.title ?? c.id}</h3>
        <p>
          {t.disk.frees(formatBytes(c.bytes))}{#if c.count}
            · {t.disk.files(c.count)}{/if}
        </p>
        <pre class="command">{c.root ? `sudo ${c.command}` : c.command}</pre>
        {#if c.root}
          <p class="hint">{t.disk.needsRoot}</p>
          <button
            class="btn btn-primary"
            onclick={() => openTerminalAt(win.serverID, undefined, `sudo ${c.command}`)}
            >{t.disk.toTerminal}</button
          >
        {:else}
          <button class="btn btn-danger" onclick={() => (confirming = c)}>{t.disk.run}</button>
        {/if}
      {:else if !isClean && current && 'path' in current}
        {@const e = current}
        <h3 class="mono">{e.path}</h3>
        <p>{formatBytes(e.size)}</p>
        <div class="actions">
          <button class="btn btn-primary" disabled={!!scanning} onclick={() => open(e.path)}
            >{t.disk.drill}</button
          >
          <button class="btn" onclick={() => openFilesAt(win.serverID, e.path)}
            >{t.disk.openFiles}</button
          >
        </div>
      {:else if !isClean}
        {@const m = mounts?.find((x) => x.path === filter)}
        {#if m}
          <h3 class="mono">{m.path}</h3>
          <p>{t.disk.used(formatBytes(m.used), formatBytes(m.total), pct(m))}</p>
          <div class="meter" class:alert={pct(m) >= 85}>
            <span style:width="{pct(m)}%"></span>
          </div>
        {/if}
        <p class="muted">{t.disk.noSelection}</p>
      {:else}
        <p class="muted">{t.disk.noSelection}</p>
      {/if}
      {#if note}<p class="note">{note}</p>{/if}
    </div>
  {/snippet}
</FilterShell>

<ConfirmCommand
  command={confirming?.command ?? null}
  title={confirming ? (t.disk.cleanups[confirming.id]?.title ?? '') : ''}
  body={confirming ? (t.disk.cleanups[confirming.id]?.body ?? '') : ''}
  action={t.disk.run}
  onclose={() => (confirming = null)}
  onconfirm={() => confirming && void clean(confirming)}
/>

<style>
  .rows {
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

  .crumb .tool {
    border: 0;
    background: none;
    padding: 3px;
    rotate: 180deg;
    display: flex;
  }

  .path {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: var(--text-sm);
  }

  .small {
    padding: 2px 10px;
    font-size: var(--text-sm);
  }

  .row {
    display: flex;
    align-items: center;
    gap: 10px;
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

  .bar span,
  .meter span {
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

  .warn {
    color: var(--warn);
    font-size: var(--text-xs);
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
    word-break: break-all;
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
    word-break: break-all;
  }

  .actions {
    display: flex;
    gap: 8px;
  }

  .meter {
    align-self: stretch;
    height: 8px;
    border-radius: 4px;
    background: var(--fill-track);
    overflow: hidden;
  }

  .meter.alert span {
    background: var(--danger);
  }

  .note {
    color: var(--fg-muted);
  }
</style>
