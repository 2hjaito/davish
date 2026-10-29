<script lang="ts">
  import './table.css'

  import { untrack } from 'svelte'

  import {
    type Container,
    type ContainerStats,
    type Image,
    imageRef,
    type Volume,
  } from '../../lib/api/docker'
  import { toUserError } from '../../lib/api/errors'
  import { formatBytes, formatPct } from '../../lib/format'
  import { t } from '../../lib/i18n'
  import { useAppMenu } from '../../lib/shell/menus.svelte'
  import { load, save } from '../../lib/stores/persist'
  import { servers } from '../../lib/stores/servers.svelte'
  import Empty from '../../lib/ui/Empty.svelte'
  import LoadState from '../../lib/ui/LoadState.svelte'
  import { supersedes } from '../../lib/stores/latest'
  import FilterShell, { type FilterItem } from '../../lib/ui/FilterShell.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import type { IconName } from '../../lib/ui/icons'
  import { getWindowContext } from '../../lib/wm/context'
  import { selectRequests } from '../../lib/wm/intents.svelte'
  import type { AppProps } from '../types'
  import { backend } from './backend'
  import ContainerInspector from './ContainerInspector.svelte'
  import ContainersTable from './ContainersTable.svelte'
  import {
    counts,
    type Filter,
    FILTERS,
    IMAGE_FILTERS,
    type ImageFilter,
    visible,
    visibleImages,
  } from './containers'
  import ImageInspector from './ImageInspector.svelte'
  import ImagesTable from './ImagesTable.svelte'
  import RunDialog from './RunDialog.svelte'
  import VolumesTable from './VolumesTable.svelte'

  let { win }: AppProps = $props()

  const ctx = getWindowContext()
  const d = t.docker

  type VolumeFilter = 'volumes' | 'unusedVolumes'
  type Any = Filter | ImageFilter | VolumeFilter
  const VOLUME_FILTERS: readonly VolumeFilter[] = ['volumes', 'unusedVolumes']
  const ALL: readonly Any[] = [...FILTERS, ...IMAGE_FILTERS, ...VOLUME_FILTERS]

  const saved = load('docker', { filter: 'all' as Any })
  let filter = $state<Any>(ALL.includes(saved.filter) ? saved.filter : 'all')
  let query = $state('')
  let containers = $state.raw<Container[] | null>(null)
  let images = $state.raw<Image[] | null>(null)
  let volumes = $state.raw<Volume[] | null>(null)
  let stats = $state.raw<Record<string, ContainerStats>>({})
  let runMode = $state<'pull' | 'run' | null>(null)
  let error = $state('')
  let loading = $state(false)
  /** The detail page over the table, Docker Desktop style. */
  let page = $state<{ kind: 'container'; name: string } | { kind: 'image'; ref: string } | null>(
    null,
  )

  $effect(() => save('docker', { filter }))

  const kind = $derived(
    (IMAGE_FILTERS as readonly Any[]).includes(filter)
      ? 'images'
      : (VOLUME_FILTERS as readonly Any[]).includes(filter)
        ? 'volumes'
        : 'containers',
  )
  const server = $derived(servers.list.find((s) => s.id === win.serverID))
  const connected = $derived(server?.state === 'connected')

  const shown = $derived(
    containers && kind === 'containers' ? visible(containers, filter as Filter, query) : [],
  )
  const shownImages = $derived(
    images && kind === 'images' ? visibleImages(images, filter as ImageFilter, query) : [],
  )
  const shownVolumes = $derived.by(() => {
    const q = query.trim().toLowerCase()
    return (volumes ?? []).filter(
      (v) =>
        (filter === 'volumes' || v.containers === 0) &&
        (!q || `${v.name} ${v.project ?? ''}`.toLowerCase().includes(q)),
    )
  })
  const pageContainer = $derived.by(() => {
    const p = page
    return p?.kind === 'container' ? containers?.find((c) => c.name === p.name) : undefined
  })
  const pageImage = $derived.by(() => {
    const p = page
    if (p?.kind !== 'image') return undefined
    return images?.find((i) => imageRef(i) === p.ref || `${i.repository}:${i.tag}` === p.ref)
  })

  // The cards above the containers: what the running ones use together.
  const totals = $derived.by(() => {
    const ss = Object.values(stats)
    return {
      cpu: ss.reduce((a, s) => a + s.cpu, 0),
      mem: ss.reduce((a, s) => a + s.memUsed, 0),
      limit: ss[0]?.memLimit ?? 0,
      running: (containers ?? []).filter((c) => c.state === 'running').length,
    }
  })

  const icons: Record<Any, IconName> = {
    all: 'docker',
    running: 'play',
    restarting: 'refresh',
    stopped: 'pause',
    images: 'archive',
    unused: 'trash',
    volumes: 'disk',
    unusedVolumes: 'trash',
  }
  const filters = $derived.by<FilterItem[]>(() => {
    const n = containers ? counts(containers) : null
    return [
      ...FILTERS.map<FilterItem>((f, i) => ({
        id: f,
        label: d.filters[f],
        icon: icons[f],
        count: n?.[f],
        alert: f === 'restarting',
        section: i === 0 ? d.containersSection : undefined,
      })),
      ...IMAGE_FILTERS.map<FilterItem>((f, i) => ({
        id: f,
        label: d.img.filters[f],
        icon: icons[f],
        count: images ? visibleImages(images, f, '').length : undefined,
        section: i === 0 ? d.img.section : undefined,
      })),
      ...VOLUME_FILTERS.map<FilterItem>((f, i) => ({
        id: f,
        label: d.vol.filters[f],
        icon: icons[f],
        count: volumes
          ? volumes.filter((v) => f === 'volumes' || v.containers === 0).length
          : undefined,
        section: i === 0 ? d.vol.section : undefined,
      })),
    ]
  })

  const attempt = supersedes()

  async function refresh() {
    // A retry replaces a load still waiting on the server.
    const current = attempt()
    loading = true
    try {
      containers = await backend.list(win.serverID)
      // Images are cheap to read; volumes need sizes (docker system df -v),
      // so they are read once their tab is open.
      images = await backend.images(win.serverID).catch(() => images)
      if (kind === 'volumes') volumes = await backend.volumes(win.serverID)
      error = ''
    } catch (e) {
      if (current()) error = toUserError(e).message
    } finally {
      if (current()) loading = false
    }
    // Sampling takes about two seconds, so the table shows first.
    if (containers?.some((c) => c.state === 'running')) {
      try {
        const all = await backend.stats(win.serverID)
        stats = Object.fromEntries(all.map((s) => [s.id, s]))
      } catch {
        // CPU and memory are extras; the table stands without them.
      }
    } else stats = {}
  }

  // The volumes tab reads its list the first time it opens.
  $effect(() => {
    if (kind !== 'volumes' || !connected) return
    untrack(() => {
      if (volumes === null)
        backend.volumes(win.serverID).then(
          (v) => (volumes = v),
          (e: unknown) => (error = toUserError(e).message),
        )
    })
  })

  // Spotlight asked for a container: open its page.
  $effect(() => {
    if (!selectRequests.list.some((r) => r.appId === 'docker' && r.serverID === win.serverID))
      return
    untrack(() => {
      const req = selectRequests.take('docker', win.serverID)
      if (!req) return
      filter = 'all'
      query = ''
      page = { kind: 'container', name: req.key }
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

  function chooseFilter(f: string) {
    filter = f as Any
    page = null
  }

  useAppMenu(
    () => (page ? [{ label: d.table.back, run: () => (page = null) }] : []),
    () => void refresh(),
  )
</script>

<FilterShell
  group={server?.name ?? ''}
  {filters}
  bind:filter={() => filter, chooseFilter}
  sidebarLabel={d.sections}
  searchLabel={kind === 'images' ? d.img.search : kind === 'volumes' ? d.vol.search : d.search}
  bind:query
  refreshing={!connected || loading}
  ready={connected && containers !== null}
  footer=""
  {error}
  onrefresh={() => void refresh()}
>
  {#snippet actions()}
    <button
      class="tool"
      aria-label={d.run.pullTitle}
      title={d.run.pullTitle}
      disabled={!connected}
      onclick={() => (runMode = 'pull')}
    >
      <Icon name="download" size={15} strokeWidth={2.2} />
    </button>
    <button
      class="tool"
      aria-label={d.run.title}
      title={d.run.title}
      disabled={!connected}
      onclick={() => (runMode = 'run')}
    >
      <Icon name="plus" size={15} strokeWidth={2.4} />
    </button>
  {/snippet}
  {#snippet placeholder()}
    {#if !connected}
      <Empty icon="server" title={d.offline} hint={d.offlineHint} />
    {:else}
      <LoadState
        serverID={win.serverID}
        {error}
        icon="docker"
        loading={d.loading}
        onretry={() => void refresh()}
      />
    {/if}
  {/snippet}
  {#snippet list()}
    <div class="dk-view">
      {#if page}
        <div class="dk-back">
          <button onclick={() => (page = null)}>
            <span style:rotate="180deg" style:display="flex"
              ><Icon name="chevronRight" size={13} /></span
            >
            {page.kind === 'container' ? d.table.containers : d.img.section}
          </button>
          <span class="crumb">/ {page.kind === 'container' ? page.name : page.ref}</span>
        </div>
        {#if page.kind === 'container' && pageContainer}
          <ContainerInspector
            serverID={win.serverID}
            container={pageContainer}
            stats={stats[pageContainer.id]}
            onchanged={() => void refresh()}
          />
        {:else if page.kind === 'image' && pageImage}
          <ImageInspector
            serverID={win.serverID}
            image={pageImage}
            onchanged={() => ((page = null), void refresh())}
          />
        {:else}
          <p class="dk-empty">{d.table.gone}</p>
        {/if}
      {:else if kind === 'containers'}
        <div class="dk-summary">
          <div class="dk-card">
            <small>{d.table.cpuTotal}</small>
            <strong>{formatPct(totals.cpu)}</strong>
            <div class="dk-meter" class:warn={totals.cpu > 85}>
              <span style:width="{Math.min(100, totals.cpu)}%"></span>
            </div>
          </div>
          <div class="dk-card">
            <small>{d.table.memTotal}</small>
            <strong
              >{formatBytes(totals.mem)}{#if totals.limit}<span class="of"
                  >{` / ${formatBytes(totals.limit)}`}</span
                >{/if}</strong
            >
            <div class="dk-meter">
              <span style:width="{totals.limit ? (totals.mem / totals.limit) * 100 : 0}%"></span>
            </div>
          </div>
          <div class="dk-card">
            <small>{d.table.runningLabel}</small>
            <strong>{totals.running} / {containers?.length ?? 0}</strong>
            <small
              >{d.table.composeCount(
                new Set((containers ?? []).map((c) => c.project).filter(Boolean)).size,
              )}</small
            >
          </div>
        </div>
        <ContainersTable
          serverID={win.serverID}
          containers={shown}
          {stats}
          onopen={(name) => (page = { kind: 'container', name })}
          onimage={(ref) => (page = { kind: 'image', ref })}
          onchanged={() => void refresh()}
        />
      {:else if kind === 'images'}
        <ImagesTable
          serverID={win.serverID}
          images={shownImages}
          all={images ?? []}
          onopen={(ref) => (page = { kind: 'image', ref })}
          onchanged={() => void refresh()}
        />
      {:else if volumes === null}
        <p class="dk-empty">{d.vol.loading}</p>
      {:else}
        <VolumesTable
          serverID={win.serverID}
          volumes={shownVolumes}
          all={volumes}
          onchanged={() =>
            backend.volumes(win.serverID).then(
              (v) => (volumes = v),
              () => {},
            )}
        />
      {/if}
    </div>
  {/snippet}
</FilterShell>

<RunDialog serverID={win.serverID} mode={runMode} onclose={() => (runMode = null)} />

<style>
  .of {
    font-size: var(--text-sm);
    font-weight: 400;
    color: var(--fg-muted);
  }
</style>
