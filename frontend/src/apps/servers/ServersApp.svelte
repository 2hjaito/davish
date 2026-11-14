<script lang="ts">
  import { onDestroy } from 'svelte'

  import { formatDuration, formatGB, formatPct, formatRateShort } from '../../lib/format'
  import { t } from '../../lib/i18n'
  import { metrics } from '../../lib/stores/metrics.svelte'
  import { notifications } from '../../lib/stores/notifications.svelte'
  import { type ConnState, servers } from '../../lib/stores/servers.svelte'
  import { ui } from '../../lib/stores/ui.svelte'
  import Meter from '../../lib/ui/charts/Meter.svelte'
  import Empty from '../../lib/ui/Empty.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import TitleBar from '../../lib/wm/TitleBar.svelte'
  import type { AppProps } from '../types'
  import ServerMap from './ServerMap.svelte'
  import { alertsFor, memPct } from './summary'

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

  /** The server whose map is open over the grid. */
  let mapOf = $state<string | null>(null)
  const mapped = $derived(servers.list.find((x) => x.id === mapOf && x.state === 'connected'))

  // Many servers: a name filter and alerts first.
  let query = $state('')
  let sort = $state<'alerts' | 'name'>('alerts')
  const many = $derived(servers.list.length >= 4)
  const shown = $derived.by(() => {
    const q = query.trim().toLowerCase()
    const list = servers.list.filter((x) => !q || `${x.name} ${x.host}`.toLowerCase().includes(q))
    if (sort === 'name') return [...list].sort((a, b) => a.name.localeCompare(b.name))
    const weight = (id: string) => alertsFor(notifications.list, id).count
    return [...list].sort((a, b) => weight(b.id) - weight(a.id))
  })

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

<TitleBar>
  <span class="title">{win.title}</span>
  <span class="count">{s.overview.connected(up, servers.list.length)}</span>
</TitleBar>

<div class="body">
  {#if mapped}
    <ServerMap serverID={mapped.id} name={mapped.name} onback={() => (mapOf = null)} />
  {:else if servers.list.length === 0}
    <Empty icon="server" title={s.none} hint={s.overview.emptyHint} />
  {:else}
    {#if many}
      <div class="tools">
        <input
          type="search"
          placeholder={s.map.filter}
          aria-label={s.map.filter}
          bind:value={query}
        />
        <div class="segmented" role="group">
          {#each ['alerts', 'name'] as const as k (k)}
            <button type="button" aria-pressed={sort === k} onclick={() => (sort = k)}
              >{s.map.sort[k]}</button
            >
          {/each}
        </div>
      </div>
    {/if}
    <div class="grid">
      {#each shown as srv (srv.id)}
        {@const m = metrics.latest(srv.id)}
        {@const al = alertsFor(notifications.list, srv.id)}
        <article class="card" class:active={srv.id === servers.activeId}>
          <header>
            <span class="dot" style:background={dotColor[srv.state]}></span>
            <span class="name">{srv.name}</span>
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
                <Icon name="globe" size={13} />{s.map.open}
              </button>
            {/if}
            {#if srv.state === 'disconnected'}
              <button class="btn" onclick={() => void servers.ensureConnected(srv.id)}
                >{s.overview.connect}</button
              >
            {:else if srv.state === 'failed'}
              <button class="btn" onclick={() => void servers.reconnect(srv.id)}
                >{s.reconnect}</button
              >
              <button class="btn" onclick={() => ui.editServer(srv.id)}>{s.edit}</button>
            {/if}
            {#if srv.id !== servers.activeId}
              <button class="btn btn-primary" onclick={() => servers.select(srv.id)}
                >{s.overview.switch}</button
              >
            {:else}
              <span class="here">{s.overview.here}</span>
            {/if}
          </footer>
        </article>
      {/each}
    </div>
    <p class="hint">{s.overview.hint}</p>
  {/if}
</div>

<style>
  .tools {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 12px;
  }

  .tools input {
    flex: 1;
    min-width: 0;
    max-width: 280px;
    padding: 4px 10px;
    border: 1px solid var(--divider-soft);
    border-radius: 8px;
    background: var(--card-bg);
    font: inherit;
    font-size: var(--text-sm);
    color: var(--fg);
  }

  .segmented {
    display: flex;
    padding: 2px;
    border-radius: 8px;
    background: var(--fill-soft);
  }

  .segmented button {
    padding: 2px 9px;
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

  .title {
    flex: 1;
    text-align: center;
    font-weight: 600;
  }

  .count {
    font-size: var(--text-xs);
    color: var(--fg-muted);
  }

  .body {
    flex: 1;
    min-height: 0;
    overflow: auto;
    padding: 16px 18px;
    background: var(--content-bg);
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
