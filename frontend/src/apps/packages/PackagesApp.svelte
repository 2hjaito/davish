<script lang="ts">
  import { untrack } from 'svelte'

  import { toUserError } from '../../lib/api/errors'
  import type { Inventory, PackageOp, ServerApp } from '../../lib/api/packages'
  import { t } from '../../lib/i18n/vi'
  import { load, save } from '../../lib/stores/persist'
  import { servers } from '../../lib/stores/servers.svelte'
  import Dialog from '../../lib/ui/Dialog.svelte'
  import Empty from '../../lib/ui/Empty.svelte'
  import FilterShell, { type FilterItem } from '../../lib/ui/FilterShell.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import StatusList, { type StatusItem } from '../../lib/ui/StatusList.svelte'
  import { getWindowContext } from '../../lib/wm/context'
  import { openTerminalAt, selectRequests } from '../../lib/wm/intents.svelte'
  import type { AppProps } from '../types'
  import { backend } from './backend'
  import PackageInspector, { type Selected } from './PackageInspector.svelte'

  type Filter = 'apps' | 'updates' | 'installed'
  const FILTERS: Filter[] = ['apps', 'updates', 'installed']
  /** Package lists older than this are worth fetching again. */
  const STALE_DAYS = 7

  let { win }: AppProps = $props()
  const ctx = getWindowContext()

  const saved = load('packages', { filter: 'apps' as Filter })
  let filter = $state<Filter>(FILTERS.includes(saved.filter) ? saved.filter : 'apps')
  let query = $state('')
  let inv = $state.raw<Inventory | null>(null)
  let apps = $state.raw<ServerApp[] | null>(null)
  let error = $state('')
  let loading = $state(false)
  let selected = $state<string | null>(null)
  let installing = $state(false)
  let installName = $state('')
  let planError = $state('')

  $effect(() => save('packages', { filter }))

  const server = $derived(servers.list.find((s) => s.id === win.serverID))
  const connected = $derived(server?.state === 'connected')
  const security = $derived(inv?.updates.filter((u) => u.security) ?? [])
  const listsDays = $derived(inv && inv.listsAge >= 0 ? Math.floor(inv.listsAge / 86400) : -1)

  type Row = StatusItem & { sel: Selected; text: string }

  const rows = $derived.by<Row[]>(() => {
    const updates = new Map((inv?.updates ?? []).map((u) => [u.name, u]))
    if (filter === 'apps')
      return (apps ?? []).map((a) => ({
        key: `app:${a.id}`,
        name: a.label,
        health: 'ok',
        note: a.version || t.packages.noVersion,
        title: a.path,
        sel: { kind: 'app', app: a },
        text: `${a.label} ${a.id} ${a.path}`,
      }))
    const pkgs = new Map((inv?.installed ?? []).map((p) => [p.name, p]))
    if (filter === 'updates')
      return [...updates.values()]
        .sort((a, b) => Number(b.security) - Number(a.security) || a.name.localeCompare(b.name))
        .map((u) => ({
          key: `pkg:${u.name}`,
          name: u.name,
          health: u.security ? 'failed' : 'busy',
          note: u.security ? `${t.packages.security} · ${u.available}` : u.available,
          sel: { kind: 'package', name: u.name, update: u, pkg: pkgs.get(u.name) },
          text: u.name,
        }))
    return [...pkgs.values()].map((p) => ({
      key: `pkg:${p.name}`,
      name: p.name,
      health: updates.has(p.name) ? 'busy' : 'ok',
      note: p.version,
      title: p.summary,
      sel: { kind: 'package', name: p.name, pkg: p, update: updates.get(p.name) },
      text: `${p.name} ${p.summary}`,
    }))
  })

  const shown = $derived.by(() => {
    const q = query.trim().toLowerCase()
    return rows.filter((r) => !q || r.text.toLowerCase().includes(q))
  })
  const current = $derived(shown.find((r) => r.key === selected)?.sel ?? null)

  const filters = $derived<FilterItem[]>(
    FILTERS.map((f) => ({
      id: f,
      label: t.packages.filters[f],
      icon: f === 'apps' ? 'grid' : f === 'updates' ? 'download' : 'package',
      count:
        f === 'apps' ? apps?.length : f === 'updates' ? inv?.updates.length : inv?.installed.length,
      alert: f === 'updates' && security.length > 0,
    })),
  )

  async function refresh() {
    if (loading || !connected) return
    loading = true
    try {
      ;[inv, apps] = await Promise.all([
        backend.inventory(win.serverID),
        backend.apps(win.serverID),
      ])
      error = ''
    } catch (e) {
      error = toUserError(e).message
    } finally {
      loading = false
    }
  }

  // Root is needed for every change, so the command goes to a terminal for
  // the user to check and run (docs/SECURITY.md).
  async function terminal(op: PackageOp, names: string[]) {
    if (!inv) return
    planError = ''
    try {
      const cmd = await backend.plan(inv.manager, op, names)
      openTerminalAt(win.serverID, undefined, cmd)
      installing = false
      installName = ''
    } catch (e) {
      planError = toUserError(e).message
    }
  }

  // Spotlight asked for a program: show it, whatever the filter.
  $effect(() => {
    if (!selectRequests.list.some((r) => r.appId === 'packages' && r.serverID === win.serverID))
      return
    untrack(() => {
      const req = selectRequests.take('packages', win.serverID)
      if (!req) return
      filter = 'apps'
      query = ''
      selected = `app:${req.key}`
    })
  })

  // Load when the server comes up and when the window regains focus; no
  // polling, as every read is a command in the audit log.
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

<FilterShell
  group={server?.name ?? ''}
  {filters}
  bind:filter={() => filter, (f) => ((filter = f as Filter), (selected = null))}
  sidebarLabel={t.packages.sections}
  searchLabel={t.packages.search}
  bind:query
  refreshing={!connected || loading}
  ready={connected && inv !== null && apps !== null}
  footer={inv ? `${t.packages.count(shown.length)} · ${t.packages.manager(inv.manager)}` : ''}
  {error}
  onrefresh={() => void refresh()}
>
  {#snippet actions()}
    <button
      class="tool"
      aria-label={t.packages.install}
      title={t.packages.install}
      disabled={!inv}
      onclick={() => {
        planError = ''
        installing = true
      }}
    >
      <Icon name="plus" size={15} strokeWidth={2.2} />
    </button>
  {/snippet}
  {#snippet placeholder()}
    {#if !connected}
      <Empty icon="server" title={t.packages.offline} hint={t.packages.offlineHint} />
    {:else}
      <Empty icon={error ? 'warning' : 'package'} title={error || t.packages.loading} />
    {/if}
  {/snippet}
  {#snippet list()}
    <StatusList
      items={shown}
      {selected}
      label={t.packages.search}
      empty={query ? t.packages.noMatch : t.packages.empty[filter]}
      onselect={(k) => (selected = k)}
    />
  {/snippet}
  {#snippet detail()}
    {#if current}
      <PackageInspector
        serverID={win.serverID}
        sel={current}
        onterminal={(op, n) => void terminal(op, n)}
      />
    {:else if filter === 'updates' && inv}
      <section class="summary">
        {#if listsDays < 0 || listsDays >= STALE_DAYS}
          <p class="warn">
            <Icon name="warning" size={14} />
            {listsDays < 0 ? t.packages.listsNever : t.packages.listsOld(listsDays)}
          </p>
        {/if}
        {#if security.length > 0}
          <p class="sec">{t.notifications.securityUpdates(security.length, server?.name ?? '')}</p>
        {/if}
        <div class="actions">
          <button type="button" class="btn" onclick={() => void terminal('refresh', [])}
            >{t.packages.refresh}</button
          >
          {#if inv.updates.length > 0}
            <button
              type="button"
              class="btn btn-primary"
              onclick={() => void terminal('upgrade-all', [])}>{t.packages.upgradeAll}</button
            >
          {/if}
          {#if security.length > 0}
            <button
              type="button"
              class="btn"
              onclick={() =>
                void terminal(
                  'upgrade-security',
                  inv?.manager === 'apt' || inv?.manager === 'apk'
                    ? security.map((u) => u.name)
                    : [],
                )}>{t.packages.upgradeSecurity}</button
            >
          {/if}
        </div>
        <p class="muted">{t.packages.terminalHint}</p>
        {#if planError}<p class="bad">{planError}</p>{/if}
      </section>
    {:else}
      <Empty icon="package" title={t.packages.noSelection} />
    {/if}
  {/snippet}
</FilterShell>

<Dialog
  open={installing}
  title={t.packages.installTitle}
  width={420}
  onclose={() => (installing = false)}
>
  <form
    id="install-package"
    onsubmit={(e) => (
      e.preventDefault(),
      installName.trim() && void terminal('install', [installName.trim()])
    )}
  >
    <label class="field">
      <span>{t.packages.installName}</span>
      <!-- svelte-ignore a11y_autofocus -->
      <input
        class="input mono"
        spellcheck="false"
        autocomplete="off"
        autofocus
        bind:value={installName}
      />
      <small>{t.packages.installHint}</small>
    </label>
    {#if planError}<p class="bad">{planError}</p>{/if}
  </form>
  {#snippet footer()}
    <span class="spacer"></span>
    <button type="button" class="btn" onclick={() => (installing = false)}>{t.common.cancel}</button
    >
    <button
      type="submit"
      form="install-package"
      class="btn btn-primary"
      disabled={!installName.trim()}>{t.packages.installAction}</button
    >
  {/snippet}
</Dialog>

<style>
  .summary {
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 16px;
  }

  .warn {
    display: flex;
    gap: 6px;
    align-items: flex-start;
    margin: 0;
    padding: 8px 10px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    color: var(--fg);
    font-size: var(--text-sm);
  }

  .warn :global(svg) {
    flex-shrink: 0;
    margin-top: 2px;
    color: var(--warn);
  }

  .sec {
    margin: 0;
    color: var(--danger);
    font-weight: 600;
  }

  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .muted {
    margin: 0;
    color: var(--fg-muted);
    font-size: var(--text-xs);
  }

  .bad {
    margin: 8px 0 0;
    color: var(--danger);
    font-size: var(--text-sm);
  }

  .spacer {
    flex: 1;
  }
</style>
