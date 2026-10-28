<script lang="ts">
  import { untrack } from 'svelte'

  import { copyText } from '../../lib/api/clipboard'
  import { toUserError } from '../../lib/api/errors'
  import type { Port, PortList } from '../../lib/api/ports'
  import { openURL } from '../../lib/api/runtime'
  import { t } from '../../lib/i18n/vi'
  import { useAppMenu } from '../../lib/shell/menus.svelte'
  import { servers } from '../../lib/stores/servers.svelte'
  import Empty from '../../lib/ui/Empty.svelte'
  import FilterShell, { type FilterItem } from '../../lib/ui/FilterShell.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import type { IconName } from '../../lib/ui/icons'
  import StatusList, { type StatusItem } from '../../lib/ui/StatusList.svelte'
  import { getWindowContext } from '../../lib/wm/context'
  import { openInBrowser } from '../../lib/wm/intents.svelte'
  import type { AppProps } from '../types'
  import { backend } from './backend'
  import { tunnels } from './tunnels.svelte'

  let { win }: AppProps = $props()

  const ctx = getWindowContext()

  type Filter = 'all' | 'public' | 'local' | 'tunnels'
  const FILTERS: Filter[] = ['all', 'public', 'local', 'tunnels']
  const icons: Record<Filter, IconName> = {
    all: 'network',
    public: 'globe2',
    local: 'lock',
    tunnels: 'link',
  }

  let filter = $state<Filter>('all')
  let query = $state('')
  let data = $state.raw<PortList | null>(null)
  let error = $state('')
  let loading = $state(false)
  let selected = $state<string | null>(null)

  // The detail's own state, reset when another port is chosen.
  let hostHeader = $state('')
  let busy = $state(false)
  let actionError = $state('')
  let probe = $state('')
  let copied = $state(false)

  tunnels.init()

  const server = $derived(servers.list.find((s) => s.id === win.serverID))
  const connected = $derived(server?.state === 'connected')
  const keyOf = (p: Port) => `${p.addr}:${p.port}`
  const tunnelOf = (p: Port) => tunnels.of(win.serverID, p.addr, p.port)

  function keep(p: Port, f: Filter): boolean {
    if (f === 'public') return !p.loopback
    if (f === 'local') return p.loopback
    if (f === 'tunnels') return tunnelOf(p) !== undefined
    return true
  }

  const shown = $derived.by(() => {
    const q = query.trim().toLowerCase()
    return (data?.ports ?? []).filter(
      (p) =>
        keep(p, filter) &&
        (!q ||
          `${p.port} ${p.addr} ${p.process ?? ''} ${p.container ?? ''}`.toLowerCase().includes(q)),
    )
  })
  const current = $derived(data?.ports.find((p) => keyOf(p) === selected) ?? null)
  const tunnel = $derived(current ? tunnelOf(current) : undefined)
  const localAddr = $derived(tunnel ? `127.0.0.1:${tunnel.localPort}` : '')

  const filters = $derived<FilterItem[]>(
    FILTERS.map((f) => ({
      id: f,
      label: t.ports.filters[f],
      icon: icons[f],
      count: data ? data.ports.filter((p) => keep(p, f)).length : undefined,
    })),
  )

  const items = $derived<StatusItem[]>(
    shown.map((p) => ({
      key: keyOf(p),
      name: `${p.port}`,
      health: tunnelOf(p) ? 'ok' : p.loopback ? 'off' : 'busy',
      note: p.container ?? p.process ?? t.ports.unknownProcess,
      title: keyOf(p),
    })),
  )

  async function refresh() {
    if (loading) return
    loading = true
    try {
      data = await backend.list(win.serverID)
      error = ''
    } catch (e) {
      error = toUserError(e).message
    } finally {
      loading = false
    }
  }

  $effect(() => {
    void selected
    untrack(() => {
      hostHeader = tunnel?.hostHeader ?? ''
      actionError = ''
      probe = ''
      copied = false
    })
  })

  // Load when the server comes up and whenever the window regains focus.
  $effect(() => {
    if (connected) untrack(() => void refresh())
  })
  let wasFocused = untrack(() => ctx.focused)
  $effect(() => {
    const f = ctx.focused
    if (f && !wasFocused && connected) untrack(() => void refresh())
    wasFocused = f
  })

  async function run(f: () => Promise<unknown>) {
    busy = true
    actionError = ''
    try {
      await f()
    } catch (e) {
      actionError = toUserError(e).message
    } finally {
      busy = false
    }
  }

  const startTunnel = (p: Port) =>
    run(() => tunnels.start(win.serverID, p.addr, p.port, hostHeader.trim()))
  const stopTunnel = (id: string) => run(() => tunnels.stop(id))
  const tryFromServer = (p: Port) =>
    run(async () => {
      probe = t.ports.probing
      probe = await backend.probe(win.serverID, p.addr, p.port, hostHeader.trim())
    })

  async function copy() {
    copied = await copyText(localAddr)
  }

  useAppMenu(
    () => [],
    () => void refresh(),
  )
</script>

<FilterShell
  group={server?.name ?? ''}
  {filters}
  bind:filter={() => filter, (f) => (filter = f as Filter)}
  sidebarLabel={t.ports.sections}
  searchLabel={t.ports.search}
  bind:query
  refreshing={!connected || loading}
  ready={connected && data !== null}
  footer={t.ports.count(shown.length)}
  {error}
  onrefresh={() => void refresh()}
>
  {#snippet placeholder()}
    {#if !connected}
      <Empty icon="server" title={t.ports.offline} hint={t.ports.offlineHint} />
    {:else}
      <Empty icon={error ? 'warning' : 'network'} title={error || t.ports.loading} />
    {/if}
  {/snippet}
  {#snippet list()}
    <StatusList
      {items}
      {selected}
      label={t.ports.search}
      empty={data?.ports.length === 0 ? t.ports.none : t.ports.noMatch}
      onselect={(k) => (selected = k)}
    />
  {/snippet}
  {#snippet detail()}
    {#if current}
      <div class="detail">
        <header>
          <h3>:{current.port}</h3>
          <p class="muted">
            {t.ports.listen(current.addr)}
            {#if current.container}
              · {t.ports.container(current.container)}
            {:else if current.process}
              · {current.process}{current.pid ? ` (PID ${current.pid})` : ''}
            {/if}
          </p>
          {#if !current.process && !data?.root}
            <p class="hint">{t.ports.needsRoot}</p>
          {/if}
          <p class="hint">{current.loopback ? t.ports.loopbackHint : t.ports.publicHint}</p>
          <div class="row">
            <button
              type="button"
              class="btn"
              onclick={() => openInBrowser(win.serverID, current.port)}
              >{t.ports.openInDavish}</button
            >
          </div>
        </header>

        <section>
          <h4>{t.ports.tunnel}</h4>
          {#if tunnel}
            <p class="active">
              <Icon name="link" size={13} />
              {t.ports.active(localAddr)}
              <span class="muted">· {t.ports.conns(tunnel.conns)}</span>
            </p>
            {#if tunnel.moved}<p class="hint">{t.ports.moved(tunnel.remotePort)}</p>{/if}
            {#if tunnel.hostHeader}<p class="hint">Host: {tunnel.hostHeader}</p>{/if}
            <div class="row">
              <button
                type="button"
                class="btn btn-primary"
                onclick={() => openURL(`http://${localAddr}/`)}>{t.ports.openBrowser}</button
              >
              <button type="button" class="btn" onclick={() => void copy()}
                >{copied ? t.ports.copied : t.ports.copy}</button
              >
              <span class="spacer"></span>
              <button
                type="button"
                class="btn"
                disabled={busy}
                onclick={() => void stopTunnel(tunnel.id)}>{t.ports.stop}</button
              >
            </div>
          {:else}
            <p class="hint">{t.ports.tunnelHint}</p>
            <label class="field">
              <span>{t.ports.hostHeader}</span>
              <input
                class="input"
                spellcheck="false"
                autocapitalize="off"
                placeholder={t.ports.hostHeaderPlaceholder}
                bind:value={hostHeader}
              />
              <small>{t.ports.hostHeaderHint}</small>
            </label>
            <div class="row">
              <button
                type="button"
                class="btn btn-primary"
                disabled={busy}
                onclick={() => void startTunnel(current)}>{t.ports.start}</button
              >
            </div>
          {/if}
        </section>

        <section>
          <h4>{t.ports.probe}</h4>
          <p class="hint">{t.ports.probeHint}</p>
          <div class="row">
            <button
              type="button"
              class="btn"
              disabled={busy}
              onclick={() => void tryFromServer(current)}>{t.ports.probe}</button
            >
          </div>
          {#if probe}<pre class="probe">{probe}</pre>{/if}
        </section>

        {#if actionError}<p class="error" role="alert">{actionError}</p>{/if}
      </div>
    {:else}
      <Empty icon="network" title={t.ports.noSelection} />
    {/if}
  {/snippet}
</FilterShell>

<style>
  .detail {
    display: flex;
    flex-direction: column;
    gap: 16px;
    padding: 16px 18px;
    overflow: auto;
    height: 100%;
    box-sizing: border-box;
  }

  h3 {
    margin: 0;
    font-size: var(--text-lg);
    font-variant-numeric: tabular-nums;
  }

  h4 {
    margin: 0 0 6px;
    font-size: var(--text-xs);
    font-weight: 600;
    text-transform: uppercase;
    color: var(--fg-label);
  }

  p {
    margin: 2px 0;
    font-size: var(--text-sm);
  }

  .muted {
    color: var(--fg-muted);
  }

  .hint {
    font-size: var(--text-xs);
    color: var(--fg-muted);
  }

  .active {
    display: flex;
    align-items: center;
    gap: 6px;
    color: var(--ok);
  }

  .row {
    display: flex;
    gap: 8px;
    margin-top: 8px;
  }

  .spacer {
    flex: 1;
  }

  .field {
    margin-top: 6px;
  }

  .probe {
    margin: 8px 0 0;
    max-height: 220px;
    overflow: auto;
    padding: 8px 10px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    white-space: pre-wrap;
    word-break: break-all;
  }

  .error {
    color: var(--danger);
  }
</style>
