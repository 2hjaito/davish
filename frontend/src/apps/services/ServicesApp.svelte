<script lang="ts">
  import { useAppMenu } from '../../lib/shell/menus.svelte'
  import { untrack } from 'svelte'

  import { toUserError } from '../../lib/api/errors'
  import type { Unit } from '../../lib/api/services'
  import { t } from '../../lib/i18n/vi'
  import { load, save } from '../../lib/stores/persist'
  import { servers } from '../../lib/stores/servers.svelte'
  import Empty from '../../lib/ui/Empty.svelte'
  import LoadState from '../../lib/ui/LoadState.svelte'
  import { supersedes } from '../../lib/stores/latest'
  import FilterShell, { type FilterItem } from '../../lib/ui/FilterShell.svelte'
  import type { IconName } from '../../lib/ui/icons'
  import StatusList, { type StatusItem } from '../../lib/ui/StatusList.svelte'
  import { getWindowContext } from '../../lib/wm/context'
  import { selectRequests } from '../../lib/wm/intents.svelte'
  import type { AppProps } from '../types'
  import { backend } from './backend'
  import ServiceInspector from './ServiceInspector.svelte'
  import { counts, type Filter, FILTERS, health, shortName, visible } from './units'

  let { win }: AppProps = $props()

  const ctx = getWindowContext()

  const saved = load('services', { filter: 'all' as Filter })
  let filter = $state<Filter>(FILTERS.includes(saved.filter) ? saved.filter : 'all')
  let query = $state('')
  let units = $state.raw<Unit[] | null>(null)
  let error = $state('')
  let loading = $state(false)
  let selected = $state<string | null>(null)

  $effect(() => save('services', { filter }))

  const server = $derived(servers.list.find((s) => s.id === win.serverID))
  const connected = $derived(server?.state === 'connected')
  const shown = $derived(units ? visible(units, filter, query) : [])
  const current = $derived(units?.find((u) => u.name === selected) ?? null)

  const icons: Record<Filter, IconName> = {
    all: 'services',
    running: 'play',
    failed: 'warning',
    stopped: 'pause',
  }
  const filters = $derived.by<FilterItem[]>(() => {
    const n = units ? counts(units) : null
    return FILTERS.map((f) => ({
      id: f,
      label: t.services.filters[f],
      icon: icons[f],
      count: n?.[f],
      alert: f === 'failed',
    }))
  })
  const items = $derived<StatusItem[]>(
    shown.map((u) => ({
      key: u.name,
      name: shortName(u.name),
      health: health(u.active),
      note: t.services.states[u.active] ?? u.active,
      title: u.description,
    })),
  )

  const attempt = supersedes()

  async function refresh() {
    // A retry replaces a load still waiting on the server.
    const current = attempt()
    loading = true
    try {
      units = await backend.list(win.serverID)
      error = ''
    } catch (e) {
      if (current()) error = toUserError(e).message
    } finally {
      if (current()) loading = false
    }
  }

  // Spotlight asked for an item: show it, whatever the filter.
  $effect(() => {
    if (!selectRequests.list.some((r) => r.appId === 'services' && r.serverID === win.serverID))
      return
    untrack(() => {
      const req = selectRequests.take('services', win.serverID)
      if (!req) return
      filter = 'all'
      query = ''
      selected = req.key
    })
  })

  // Load when the server comes up, and again whenever the window regains
  // focus, since services change behind our back. There is no polling: every
  // list is a command in the audit log. refresh reads state of its own,
  // which must not become a dependency.
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
  sidebarLabel={t.services.sections}
  searchLabel={t.services.search}
  bind:query
  refreshing={!connected || loading}
  ready={connected && units !== null}
  footer={t.services.count(shown.length)}
  {error}
  onrefresh={() => void refresh()}
>
  {#snippet placeholder()}
    {#if !connected}
      <Empty icon="server" title={t.services.offline} hint={t.services.offlineHint} />
    {:else}
      <LoadState
        serverID={win.serverID}
        {error}
        icon="services"
        loading={t.services.loading}
        onretry={() => void refresh()}
      />
    {/if}
  {/snippet}
  {#snippet list()}
    <StatusList
      {items}
      {selected}
      label={t.services.search}
      empty={t.services.noMatch}
      onselect={(n) => (selected = n)}
    />
  {/snippet}
  {#snippet detail()}
    {#if current}
      <ServiceInspector serverID={win.serverID} unit={current} onchanged={() => void refresh()} />
    {:else}
      <Empty icon="services" title={t.services.noSelection} />
    {/if}
  {/snippet}
</FilterShell>
