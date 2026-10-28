<script lang="ts">
  import { useAppMenu } from '../../lib/shell/menus.svelte'
  import { untrack } from 'svelte'

  import { type Container, type ContainerStats, type Image, imageRef } from '../../lib/api/docker'
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
  import ImageInspector from './ImageInspector.svelte'
  import {
    counts,
    type Filter,
    FILTERS,
    health,
    IMAGE_FILTERS,
    type ImageFilter,
    visible,
    visibleImages,
  } from './containers'

  let { win }: AppProps = $props()

  const ctx = getWindowContext()

  type Any = Filter | ImageFilter
  const ALL: readonly Any[] = [...FILTERS, ...IMAGE_FILTERS]
  const saved = load('docker', { filter: 'all' as Any })
  let filter = $state<Any>(ALL.includes(saved.filter) ? saved.filter : 'all')
  let images = $state.raw<Image[] | null>(null)
  const imageMode = $derived((IMAGE_FILTERS as readonly Any[]).includes(filter))
  let query = $state('')
  let containers = $state.raw<Container[] | null>(null)
  let stats = $state.raw<Record<string, ContainerStats>>({})
  let error = $state('')
  let loading = $state(false)
  let selected = $state<string | null>(null)

  $effect(() => save('docker', { filter }))

  const server = $derived(servers.list.find((s) => s.id === win.serverID))
  const connected = $derived(server?.state === 'connected')
  const shown = $derived(
    containers && !imageMode ? visible(containers, filter as Filter, query) : [],
  )
  const current = $derived(containers?.find((c) => c.name === selected) ?? null)
  const shownImages = $derived(
    images && imageMode ? visibleImages(images, filter as ImageFilter, query) : [],
  )
  const currentImage = $derived(images?.find((i) => imageRef(i) === selected) ?? null)

  const icons: Record<Filter, IconName> = {
    all: 'docker',
    running: 'play',
    restarting: 'refresh',
    stopped: 'pause',
  }
  const imageIcons: Record<ImageFilter, IconName> = { images: 'archive', unused: 'trash' }
  const filters = $derived.by<FilterItem[]>(() => {
    const n = containers ? counts(containers) : null
    return FILTERS.map<FilterItem>((f) => ({
      id: f,
      label: t.docker.filters[f],
      icon: icons[f],
      count: n?.[f],
      alert: f === 'restarting',
    })).concat(
      IMAGE_FILTERS.map((f, i) => ({
        id: f,
        label: t.docker.img.filters[f],
        icon: imageIcons[f],
        count: images ? visibleImages(images, f, '').length : undefined,
        section: i === 0 ? t.docker.img.section : undefined,
      })),
    )
  })

  function note(c: Container): string {
    const s = stats[c.id]
    if (c.state === 'running' && s) return `${formatPct(s.cpu)} · ${formatBytes(s.memUsed)}`
    return t.docker.states[c.state] ?? c.state
  }

  const imageItems = $derived<StatusItem[]>(
    shownImages.map((i) => ({
      key: imageRef(i),
      name: i.dangling ? t.docker.img.untagged : `${i.repository}:${i.tag}`,
      health: i.containers > 0 ? 'ok' : 'off',
      note: formatBytes(i.size),
      title: `${i.id.replace(/^sha256:/, '').slice(0, 12)} · ${i.createdSince}`,
    })),
  )

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
      // Images change less often but are cheap; one refresh reads both.
      images = await backend.images(win.serverID).catch(() => images)
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
  bind:filter={() => filter, (f) => ((filter = f as Any), (selected = null))}
  sidebarLabel={t.docker.sections}
  searchLabel={imageMode ? t.docker.img.search : t.docker.search}
  bind:query
  refreshing={!connected || loading}
  ready={connected && containers !== null}
  footer={imageMode ? t.docker.img.count(shownImages.length) : t.docker.count(shown.length)}
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
    {#if imageMode}
      <StatusList
        items={imageItems}
        {selected}
        label={t.docker.search}
        empty={images?.length === 0 ? t.docker.img.none : t.docker.img.noMatch}
        onselect={(n) => (selected = n)}
      />
    {:else}
      <StatusList
        {items}
        {selected}
        label={t.docker.search}
        empty={containers?.length === 0 ? t.docker.noContainers : t.docker.noMatch}
        onselect={(n) => (selected = n)}
      />
    {/if}
  {/snippet}
  {#snippet detail()}
    {#if imageMode && currentImage}
      <ImageInspector
        serverID={win.serverID}
        image={currentImage}
        onchanged={() => ((selected = null), void refresh())}
      />
    {:else if imageMode}
      <Empty icon="archive" title={t.docker.img.noSelection} />
    {:else if current}
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
