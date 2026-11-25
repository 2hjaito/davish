<script lang="ts">
  import { onDestroy } from 'svelte'
  import { SvelteSet } from 'svelte/reactivity'

  import { formatDuration, formatGB, formatPct, formatRateShort } from '../../lib/format'
  import { t } from '../../lib/i18n'
  import { metrics } from '../../lib/stores/metrics.svelte'
  import { notifications } from '../../lib/stores/notifications.svelte'
  import { load, save } from '../../lib/stores/persist'
  import { type ConnState, type Server, servers } from '../../lib/stores/servers.svelte'
  import { ui } from '../../lib/stores/ui.svelte'
  import Meter from '../../lib/ui/charts/Meter.svelte'
  import Empty from '../../lib/ui/Empty.svelte'
  import FilterShell, { type FilterItem } from '../../lib/ui/FilterShell.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import TagChip from '../../lib/ui/TagChip.svelte'
  import {
    activeFilter,
    groupFilter,
    groupNames,
    matches,
    type Section,
    sections,
    stateOf,
    tagFilter,
    tagNames,
    ungroupedCount,
  } from '../../lib/shell/groups'
  import MultiRun from './MultiRun.svelte'
  import ServerMap from './ServerMap.svelte'
  import { alertsFor, memPct } from './summary'
  import type { AppProps } from '../types'

  // Every app takes its window; this one has nothing to read from it.
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  let { win }: AppProps = $props()

  const s = t.servers

  const dotColor: Record<ConnState, string> = {
    connected: 'var(--ok)',
    connecting: 'var(--warn)',
    reconnecting: 'var(--warn)',
    disconnected: 'var(--offline)',
    failed: 'var(--danger)',
  }

  const up = $derived(servers.list.filter((x) => x.state === 'connected').length)

  /** Running a command on several servers, in place of the grid. */
  let multi = $state(false)

  /** The server whose map is open over the grid. */
  let mapOf = $state<string | null>(null)
  const mapped = $derived(servers.list.find((x) => x.id === mapOf && x.state === 'connected'))

  // Cards show the readings; the list fits many servers on one screen (#178).
  let view = $state<'cards' | 'list'>(
    load<string>('serversView', 'cards') === 'list' ? 'list' : 'cards',
  )
  function setView(v: 'cards' | 'list') {
    view = v
    save('serversView', v)
  }

  /** Any state with a live or pending connection can be ended. */
  const linked = (st: ConnState) =>
    st === 'connected' || st === 'connecting' || st === 'reconnecting'

  // The sidebar filters by connection state, group or tag; the search also
  // matches names, hosts, tags and groups, and servers with alerts come first.
  let query = $state('')
  let sort = $state<'alerts' | 'name'>('alerts')
  const many = $derived(servers.list.length >= 4)
  let chosen = $state(load<string>('serversFilter', 'all'))
  const filter = $derived(activeFilter(servers.list, chosen))
  function chooseFilter(id: string) {
    chosen = id
    save('serversFilter', id)
  }

  const shown = $derived.by(() => {
    const q = query.trim().toLowerCase()
    const list = servers.list.filter(
      (x) =>
        matches(x, filter) &&
        (!q || `${x.name} ${x.host} ${x.tag ?? ''} ${x.group ?? ''}`.toLowerCase().includes(q)),
    )
    if (sort === 'name') return [...list].sort((a, b) => a.name.localeCompare(b.name))
    const weight = (id: string) => alertsFor(notifications.list, id).count
    return [...list].sort((a, b) => weight(b.id) - weight(a.id))
  })

  // Folders keep the sidebar's order, whatever order the servers inside take.
  const folders = $derived(
    sections(
      shown,
      groupNames(servers.list).map((g) => g.name),
    ),
  )
  const grouped = $derived(folders.some((f) => f.name !== ''))

  // Folded folders are remembered; a folder is open until the user closes it.
  const foldedKey = (name: string) => name.toLowerCase()
  const folded = new SvelteSet<string>(load<string[]>('serversFolded', []))
  function toggleFolder(name: string) {
    const k = foldedKey(name)
    if (folded.has(k)) folded.delete(k)
    else folded.add(k)
    save('serversFolded', [...folded])
  }

  const filters = $derived.by<FilterItem[]>(() => {
    const list = servers.list
    const f = s.overview.filters
    const count = (st: 'connected' | 'failed' | 'disconnected') =>
      list.filter((x) => stateOf(x.state) === st).length
    const out: FilterItem[] = [
      { id: 'all', label: f.all, icon: 'grid', count: list.length },
      { id: 'connected', label: f.connected, icon: 'check', count: count('connected') },
      { id: 'failed', label: f.failed, icon: 'warning', count: count('failed'), alert: true },
      { id: 'disconnected', label: f.disconnected, icon: 'power', count: count('disconnected') },
    ]
    const gs = groupNames(list)
    gs.forEach((g, i) =>
      out.push({
        id: groupFilter(g.name),
        label: g.name,
        icon: 'folder',
        count: g.count,
        section: i === 0 ? f.groups : undefined,
      }),
    )
    if (gs.length > 0 && ungroupedCount(list) > 0)
      out.push({ id: 'ungrouped', label: f.noGroup, icon: 'server', count: ungroupedCount(list) })
    tagNames(list).forEach((n, i) =>
      out.push({
        id: tagFilter(n.name),
        label: n.name,
        icon: 'tag',
        count: n.count,
        section: i === 0 ? f.tags : undefined,
      }),
    )
    return out
  })

  /** What a folder's header shows: how many are up, and the alerts inside. */
  function folderStat(sec: Section<Server>) {
    const up = sec.servers.filter((x) => x.state === 'connected').length
    const als = sec.servers.map((x) => alertsFor(notifications.list, x.id))
    const count = als.reduce((n, a) => n + a.count, 0)
    const level = als.some((a) => a.level === 'error') ? 'error' : 'warn'
    return { up, count, level }
  }

  // Readings flow only while this window is open, one collector per
  // connected server; the menu bar's own watch on the active one stays.
  // eslint-disable-next-line svelte/prefer-svelte-reactivity -- bookkeeping, never rendered
  const watching = new Map<string, () => void>()
  $effect(() => {
    const ids = new Set(servers.list.filter((x) => x.state === 'connected').map((x) => x.id))
    for (const [id, off] of watching)
      if (!ids.has(id)) {
        off()
        watching.delete(id)
      }
    for (const id of ids) if (!watching.has(id)) watching.set(id, metrics.watch(id))
  })
  onDestroy(() => {
    for (const off of watching.values()) off()
  })
</script>

{#snippet row(srv: Server)}
  {@const m = metrics.latest(srv.id)}
  {@const al = alertsFor(notifications.list, srv.id)}
  {@const live = srv.state === 'connected' && m}
  <tr class:active={srv.id === servers.activeId}>
    <td
      ><div class="who">
        <span class="dot" style:background={dotColor[srv.state]}></span>
        <button class="link sname" title={s.overview.switch} onclick={() => servers.select(srv.id)}
          >{srv.name}</button
        >
        {#if srv.tag}<TagChip tag={srv.tag} color={srv.color} />{/if}
        <span class="host">{srv.user}@{srv.host}</span>
        {#if al.count}
          <span class="badge {al.level}" title={al.latest}>
            <Icon name="warning" size={11} strokeWidth={2.4} />{al.count}
          </span>
        {/if}
      </div></td
    >
    {#if live}
      <td class="num">{formatPct(m.cpu)}</td>
      <td class="num">{formatPct(memPct(m.memUsed, m.memTotal))}</td>
      <td class="num">{formatPct(m.diskPct)}</td>
      <td class="num">{m.load1.toFixed(2)}</td>
    {:else}
      <td class="muted" colspan="4">{s.status[srv.state]}</td>
    {/if}
    <td class="acts">
      {#if srv.state === 'connected'}
        <button
          class="icon"
          title={s.map.open}
          aria-label={s.map.open}
          onclick={() => (mapOf = srv.id)}><Icon name="globe" size={13} /></button
        >
      {/if}
      {#if linked(srv.state)}
        <button
          class="icon"
          title={s.overview.disconnect}
          aria-label={s.overview.disconnect}
          onclick={() => void servers.disconnect(srv.id)}><Icon name="power" size={13} /></button
        >
      {:else}
        <button
          class="icon"
          title={s.overview.connect}
          aria-label={s.overview.connect}
          onclick={() => void servers.reconnect(srv.id)}><Icon name="refresh" size={13} /></button
        >
      {/if}
      <button class="icon" title={s.edit} aria-label={s.edit} onclick={() => ui.editServer(srv.id)}
        ><Icon name="editor" size={13} /></button
      >
    </td>
  </tr>
{/snippet}

{#snippet card(srv: Server)}
  {@const m = metrics.latest(srv.id)}
  {@const al = alertsFor(notifications.list, srv.id)}
  <article class="card" class:active={srv.id === servers.activeId}>
    <header>
      <span class="dot" style:background={dotColor[srv.state]}></span>
      <span class="name">{srv.name}</span>
      {#if srv.tag}<TagChip tag={srv.tag} color={srv.color} />{/if}
      {#if al.count}
        <span class="badge {al.level}" title={al.latest}>
          <Icon name="warning" size={11} strokeWidth={2.4} />{al.count}
        </span>
      {/if}
    </header>
    <p class="host">{srv.user}@{srv.host}</p>

    {#if srv.state === 'connected' && m}
      <dl>
        <dt>{s.overview.cpu}</dt>
        <dd><Meter pct={m.cpu} label={s.overview.cpu} /></dd>
        <dd class="val">{formatPct(m.cpu)}</dd>
        <dt>{s.overview.ram}</dt>
        <dd>
          <Meter pct={memPct(m.memUsed, m.memTotal)} label={s.overview.ram} />
        </dd>
        <dd class="val">{formatGB(m.memUsed)} / {formatGB(m.memTotal)} GB</dd>
        <dt>{s.overview.disk}</dt>
        <dd><Meter pct={m.diskPct} label={s.overview.disk} /></dd>
        <dd class="val">{formatPct(m.diskPct)}</dd>
      </dl>
      <p class="facts">
        <span>{s.overview.load} {m.load1.toFixed(2)}</span>
        <span>{s.overview.uptime} {formatDuration(m.uptime, t.units)}</span>
        <span>↓ {formatRateShort(m.rxBps)} ↑ {formatRateShort(m.txBps)}</span>
      </p>
    {:else if srv.state === 'connected'}
      <p class="state">{s.overview.waiting}</p>
    {:else if srv.state === 'failed'}
      <button
        class="state failed why"
        title={t.connError.details}
        onclick={() => servers.showFailure(srv.id)}
      >
        {s.status.failed}{#if srv.error}: {t.errors[srv.error]}{/if}
        <span class="more">{t.connError.details} ›</span>
      </button>
    {:else}
      <p class="state">{s.status[srv.state]}</p>
    {/if}

    {#if al.latest}
      <p class="alert {al.level}"><Icon name="warning" size={12} />{al.latest}</p>
    {/if}

    <footer>
      {#if srv.state === 'connected'}
        <button class="btn" onclick={() => (mapOf = srv.id)}>
          <Icon name="globe" size={13} />
          {s.map.open}
        </button>
      {/if}
      {#if srv.state === 'disconnected'}
        <button class="btn" onclick={() => void servers.ensureConnected(srv.id)}
          >{s.overview.connect}</button
        >
      {:else if srv.state === 'failed'}
        <button class="btn" onclick={() => void servers.reconnect(srv.id)}>{s.reconnect}</button>
      {:else if linked(srv.state)}
        <button class="btn" onclick={() => void servers.disconnect(srv.id)}
          >{s.overview.disconnect}</button
        >
      {/if}
      <button class="btn" onclick={() => ui.editServer(srv.id)}>{s.edit}</button>
      {#if srv.id !== servers.activeId}
        <button class="btn btn-primary" onclick={() => servers.select(srv.id)}
          >{s.overview.switch}</button
        >
      {:else}
        <span class="here">{s.overview.here}</span>
      {/if}
    </footer>
  </article>
{/snippet}

{#snippet folderHead(sec: Section<Server>)}
  {@const st = folderStat(sec)}
  {@const open = !folded.has(foldedKey(sec.name))}
  <button
    type="button"
    class="fold"
    aria-expanded={open}
    aria-label={open ? s.overview.fold : s.overview.unfold}
    onclick={() => toggleFolder(sec.name)}
  >
    <span class="chev" class:open><Icon name="chevronRight" size={12} strokeWidth={2.6} /></span>
    <Icon name="folder" size={14} />
    <span class="fname">{sec.name || s.overview.filters.noGroup}</span>
    <span class="fcount">{st.up}/{sec.servers.length}</span>
    {#if st.count}
      <span class="badge {st.level}">
        <Icon name="warning" size={11} strokeWidth={2.4} />{st.count}
      </span>
    {/if}
  </button>
{/snippet}

{#snippet pageView()}
  {#if multi}
    <MultiRun onback={() => (multi = false)} />
  {:else if mapped}
    <ServerMap serverID={mapped.id} name={mapped.name} onback={() => (mapOf = null)} />
  {/if}
{/snippet}

{#snippet listView()}
  <div class="content">
    {#if view === 'list'}
      <table class="list">
        <thead>
          <tr>
            <th>{s.overview.server}</th>
            <th>{s.overview.cpu}</th>
            <th>{s.overview.ram}</th>
            <th>{s.overview.disk}</th>
            <th>{s.overview.load}</th>
            <th></th>
          </tr>
        </thead>
        {#each folders as sec (sec.name)}
          <tbody>
            {#if grouped}
              <tr class="grp"><td colspan="6">{@render folderHead(sec)}</td></tr>
            {/if}
            {#if !folded.has(foldedKey(sec.name))}
              {#each sec.servers as srv (srv.id)}
                {@render row(srv)}
              {/each}
            {/if}
          </tbody>
        {/each}
      </table>
    {:else}
      {#each folders as sec (sec.name)}
        {#if grouped}<div class="gh">{@render folderHead(sec)}</div>{/if}
        {#if !folded.has(foldedKey(sec.name))}
          <div class="grid">
            {#each sec.servers as srv (srv.id)}
              {@render card(srv)}
            {/each}
          </div>
        {/if}
      {/each}
    {/if}
    <p class="hint">{s.overview.hint}</p>
  </div>
{/snippet}

<FilterShell
  group=""
  {filters}
  bind:filter={() => filter, chooseFilter}
  sidebarLabel={s.overview.filters.sidebar}
  searchLabel={s.map.filter}
  bind:query
  refreshing={false}
  ready={servers.list.length > 0}
  footer={s.overview.connected(up, servers.list.length)}
  page={multi || mapped ? pageView : undefined}
  pageTitle={multi ? s.run.open : (mapped?.name ?? '')}
  onrefresh={() => void servers.reload()}
  list={listView}
>
  {#snippet actions()}
    {#if !multi && !mapped}
      {#if many}
        <div class="segmented" role="group">
          {#each ['alerts', 'name'] as const as k (k)}
            <button type="button" aria-pressed={sort === k} onclick={() => (sort = k)}
              >{s.map.sort[k]}</button
            >
          {/each}
        </div>
      {/if}
      <div class="segmented" role="group" aria-label={s.overview.view}>
        {#each ['cards', 'list'] as const as v (v)}
          <button
            type="button"
            aria-pressed={view === v}
            aria-label={s.overview.views[v]}
            title={s.overview.views[v]}
            onclick={() => setView(v)}
          >
            <Icon name={v === 'cards' ? 'viewCards' : 'viewTable'} size={14} strokeWidth={2.2} />
          </button>
        {/each}
      </div>
      <button class="tool" title={s.add} aria-label={s.add} onclick={() => ui.addServer()}>
        <Icon name="plus" size={15} strokeWidth={2.2} />
      </button>
      <button
        class="tool"
        title={s.import}
        aria-label={s.import}
        onclick={() => (ui.importDialog = true)}
      >
        <Icon name="download" size={15} strokeWidth={2.2} />
      </button>
      <button
        class="tool"
        title={s.run.open}
        aria-label={s.run.open}
        disabled={up === 0}
        onclick={() => (multi = true)}
      >
        <Icon name="terminal" size={15} strokeWidth={2.2} />
      </button>
    {/if}
  {/snippet}
  {#snippet placeholder()}
    <Empty icon="server" title={s.none} hint={s.overview.emptyHint} />
  {/snippet}
</FilterShell>

<style>
  /* FilterShell's list does not scroll by itself; this is what scrolls, above its footer. */
  .content {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    padding: 16px 18px;
  }

  /* A folder header: fold arrow, folder, name, how many are up, alerts. */
  .fold {
    display: flex;
    align-items: center;
    gap: 7px;
    width: 100%;
    padding: 4px 2px;
    border: 0;
    background: none;
    font: inherit;
    font-weight: 600;
    color: var(--fg);
    text-align: left;
  }

  .fold:hover .fname {
    color: var(--accent);
  }

  .chev {
    display: inline-flex;
    color: var(--fg-muted);
    transition: transform var(--dur-fast) var(--ease-out);
  }

  .chev.open {
    transform: rotate(90deg);
  }

  .fname {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .fcount {
    font-size: var(--text-xs);
    font-weight: 400;
    color: var(--fg-muted);
    font-variant-numeric: tabular-nums;
  }

  .gh {
    margin: 14px 0 8px;
  }

  .gh:first-child {
    margin-top: 0;
  }

  .list .grp td {
    padding: 10px 8px 4px;
    background: none;
  }

  .list {
    width: 100%;
    border-collapse: collapse;
    font-size: var(--text-sm);
  }

  .list th {
    padding: 4px 8px;
    white-space: nowrap;
    text-align: left;
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--fg-muted);
    border-bottom: 1px solid var(--divider-soft);
  }

  .list td {
    padding: 5px 8px;
    border-bottom: 1px solid var(--divider-soft);
    white-space: nowrap;
  }

  .list tr.active td {
    background: var(--selection-inactive);
  }

  .list .who {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .list th:first-child,
  .list td:first-child {
    width: 100%;
  }

  .list .sname {
    min-width: 120px;
    text-align: left;
  }

  .list .host {
    color: var(--fg-muted);
    font-size: var(--text-xs);
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .list .num {
    font-variant-numeric: tabular-nums;
  }

  .list .muted {
    color: var(--fg-muted);
    font-size: var(--text-xs);
  }

  .list .acts {
    text-align: right;
  }

  .link {
    border: 0;
    background: none;
    padding: 0;
    font: inherit;
    font-weight: 600;
    color: var(--fg);
    cursor: pointer;
  }

  .link:hover {
    color: var(--accent);
  }

  .icon {
    display: inline-flex;
    padding: 3px 5px;
    border: 0;
    border-radius: 6px;
    background: none;
    color: var(--fg-muted);
  }

  .icon:hover {
    background: var(--fill-soft);
    color: var(--fg);
  }

  .segmented {
    flex-shrink: 0;
    display: flex;
    margin-right: 6px;
    padding: 2px;
    border-radius: 8px;
    background: var(--fill-soft);
  }

  .segmented button {
    display: flex;
    padding: 3px 9px;
    border: 0;
    border-radius: 6px;
    background: none;
    font-size: var(--text-xs);
    color: var(--fg-muted);
  }

  .segmented button[aria-pressed='true'] {
    background: var(--chart-surface);
    color: var(--fg);
    font-weight: 600;
    box-shadow: var(--switch-knob-shadow);
  }

  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: 12px;
  }

  .card {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 12px 14px;
    border-radius: var(--card-radius);
    background: var(--card-bg);
    border: 1px solid var(--divider-soft);
    min-width: 0;
  }

  .card.active {
    border-color: var(--accent);
  }

  header {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .name {
    flex: 1;
    min-width: 0;
    font-size: var(--text-lg);
    font-weight: 600;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .badge {
    display: flex;
    align-items: center;
    gap: 3px;
    padding: 1px 6px;
    border-radius: 9px;
    font-size: var(--text-xs);
    font-variant-numeric: tabular-nums;
    background: var(--fill-soft);
    color: var(--fg-muted);
  }

  .badge.warn {
    background: color-mix(in srgb, var(--warn) 16%, transparent);
    color: var(--warn);
  }

  .badge.error {
    background: color-mix(in srgb, var(--danger) 14%, transparent);
    color: var(--danger);
  }

  .host {
    margin: -6px 0 0 16px;
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    color: var(--fg-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  dl {
    margin: 0;
    display: grid;
    grid-template-columns: 3.5em minmax(0, 1fr) auto;
    align-items: center;
    gap: 6px 10px;
    font-size: var(--text-sm);
  }

  dt {
    color: var(--fg-muted);
    font-size: var(--text-xs);
  }

  dd {
    margin: 0;
  }

  .val {
    text-align: right;
    font-variant-numeric: tabular-nums;
    font-size: var(--text-xs);
  }

  .facts {
    margin: 0;
    display: flex;
    flex-wrap: wrap;
    gap: 4px 12px;
    font-size: var(--text-xs);
    color: var(--fg-muted);
    font-variant-numeric: tabular-nums;
  }

  .state {
    margin: 4px 0;
    font-size: var(--text-sm);
    color: var(--fg-muted);
  }

  .state.failed {
    color: var(--danger);
  }

  /* The failure line opens the error dialog, where the cause can be copied. */
  .why {
    display: block;
    width: 100%;
    padding: 0;
    border: 0;
    background: none;
    text-align: left;
    font: inherit;
    font-size: var(--text-sm);
    line-height: 1.4;
    cursor: pointer;
  }

  .why .more {
    margin-left: 4px;
    color: var(--accent);
    white-space: nowrap;
  }

  .why:hover .more {
    text-decoration: underline;
  }

  .alert {
    margin: 0;
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: var(--text-xs);
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
  }

  .alert :global(svg) {
    flex-shrink: 0;
    color: var(--fg-muted);
  }

  .alert.warn :global(svg) {
    color: var(--warn);
  }

  .alert.error :global(svg) {
    color: var(--danger);
  }

  footer {
    margin-top: auto;
    display: flex;
    flex-wrap: wrap;
    justify-content: flex-end;
    align-items: center;
    gap: 6px;
  }

  footer .btn {
    white-space: nowrap;
  }

  .here {
    font-size: var(--text-xs);
    color: var(--fg-muted);
  }

  .hint {
    margin: 14px 0 0;
    font-size: var(--text-xs);
    color: var(--fg-subtle);
  }
</style>
