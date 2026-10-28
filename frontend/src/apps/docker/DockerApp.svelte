<script lang="ts">
  import { useAppMenu } from '../../lib/shell/menus.svelte'
  import { untrack } from 'svelte'

  import type { Container, ContainerStats } from '../../lib/api/docker'
  import { toUserError } from '../../lib/api/errors'
  import { formatBytes, formatPct } from '../../lib/format'
  import { t } from '../../lib/i18n/vi'
  import { load, save } from '../../lib/stores/persist'
  import { servers } from '../../lib/stores/servers.svelte'
  import Empty from '../../lib/ui/Empty.svelte'
  import FilterShell, { type FilterItem } from '../../lib/ui/FilterShell.svelte'
  import type { IconName } from '../../lib/ui/icons'
  import StatusList, { type StatusItem } from '../../lib/ui/StatusList.svelte'
  import { getWindowContext } from '../../lib/wm/context'
  import { selectRequests } from '../../lib/wm/intents.svelte'
  import type { AppProps } from '../types'
  import { backend } from './backend'
  import ContainerInspector from './ContainerInspector.svelte'
  import { counts, type Filter, FILTERS, health, visible } from './containers'

  let { win }: AppProps = $props()

  const ctx = getWindowContext()

  const saved = load('docker', { filter: 'all' as Filter })
  let filter = $state<Filter>(FILTERS.includes(saved.filter) ? saved.filter : 'all')
  let query = $state('')
  let containers = $state.raw<Container[] | null>(null)
  let stats = $state.raw<Record<string, ContainerStats>>({})
  let error = $state('')
  let loading = $state(false)
  let selected = $state<string | null>(null)

  $effect(() => save('docker', { filter }))

  const server = $derived(servers.list.find((s) => s.id === win.serverID))
  const connected = $derived(server?.state === 'connected')
  const shown = $derived(containers ? visible(containers, filter, query) : [])
  const current = $derived(containers?.find((c) => c.name === selected) ?? null)

  const icons: Record<Filter, IconName> = {
    all: 'docker',
    running: 'play',
    restarting: 'refresh',
    stopped: 'pause',
  }
  const filters = $derived.by<FilterItem[]>(() => {
    const n = containers ? counts(containers) : null
    return FILTERS.map((f) => ({
      id: f,
      label: t.docker.filters[f],
      icon: icons[f],
      count: n?.[f],
      alert: f === 'restarting',
    }))
  })

  function note(c: Container): string {
    const s = stats[c.id]
    if (c.state === 'running' && s) return `${formatPct(s.cpu)} · ${formatBytes(s.memUsed)}`
    return t.docker.states[c.state] ?? c.state
  }

  const items = $derived<StatusItem[]>(
    shown.map((c) => ({
      key: c.name,
      name: c.name,
      health: health(c.state),
      note: note(c),
      title: `${c.image} · ${c.status}`,
    })),
  )

  async function refresh() {
    if (loading) return
    loading = true
    try {
      containers = await backend.list(win.serverID)
      error = ''
    } catch (e) {
      error = toUserError(e).message
    } finally {
      loading = false
    }
    // Sampling takes about two seconds, so the list shows first.
    if (containers?.some((c) => c.state === 'running')) {
      try {
        const all = await backend.stats(win.serverID)
        stats = Object.fromEntries(all.map((s) => [s.id, s]))
      } catch {
        // CPU and memory are extras; the list stands without them.
      }
    }
  }

  // Spotlight asked for an item: show it, whatever the filter.
  $effect(() => {
    if (!selectRequests.list.some((r) => r.appId === 'docker' && r.serverID === win.serverID))
      return
    untrack(() => {
      const req = selectRequests.take('docker', win.serverID)
      if (!req) return
      filter = 'all'
      query = ''
      selected = req.key
    })
  })

  // Load when the server comes up and whenever the window regains focus.
  // No polling: every read is a command in the audit log.
  $effect(() => {
    if (connected) untrack(() => void refresh())
  })
  let wasFocused = untrack(() => ctx.focused)
  $effect(() => {
    const f = ctx.focused
    if (f && !wasFocused && connected) untrack(() => void refresh())
    wasFocused = f
  })

  useAppMenu(
    () => [],
    () => void refresh(),
  )
</script>

<FilterShell
  group={server?.name ?? ''}
  {filters}
  bind:filter={() => filter, (f) => (filter = f as Filter)}
  sidebarLabel={t.docker.sections}
  searchLabel={t.docker.search}
  bind:query
  refreshing={!connected || loading}
  ready={connected && containers !== null}
  footer={t.docker.count(shown.length)}
  {error}
  onrefresh={() => void refresh()}
>
  {#snippet placeholder()}
    {#if !connected}
      <Empty icon="server" title={t.docker.offline} hint={t.docker.offlineHint} />
    {:else}
      <Empty icon={error ? 'warning' : 'docker'} title={error || t.docker.loading} />
    {/if}
  {/snippet}
  {#snippet list()}
    <StatusList
      {items}
      {selected}
      label={t.docker.search}
      empty={containers?.length === 0 ? t.docker.noContainers : t.docker.noMatch}
      onselect={(n) => (selected = n)}
    />
  {/snippet}
  {#snippet detail()}
    {#if current}
      <ContainerInspector
        serverID={win.serverID}
        container={current}
        stats={stats[current.id]}
        onchanged={() => void refresh()}
      />
    {:else}
      <Empty icon="docker" title={t.docker.noSelection} />
    {/if}
  {/snippet}
</FilterShell>
