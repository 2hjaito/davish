<script lang="ts">
  import { untrack } from 'svelte'

  import type { CronChange, CronJob, CronOp, CronPlan, CronTable } from '../../lib/api/cron'
  import { toUserError } from '../../lib/api/errors'
  import { t } from '../../lib/i18n/vi'
  import { load, save } from '../../lib/stores/persist'
  import { servers } from '../../lib/stores/servers.svelte'
  import Empty from '../../lib/ui/Empty.svelte'
  import FilterShell, { type FilterItem } from '../../lib/ui/FilterShell.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import StatusList, { type StatusItem } from '../../lib/ui/StatusList.svelte'
  import { getWindowContext } from '../../lib/wm/context'
  import type { AppProps } from '../types'
  import { backend } from './backend'
  import ChangeDialog from './ChangeDialog.svelte'
  import CronInspector from './CronInspector.svelte'
  import { describe } from './describe'
  import JobDialog from './JobDialog.svelte'
  import { formatRun } from './when'

  type Filter = 'mine' | 'system' | 'disabled'
  const FILTERS: Filter[] = ['mine', 'system', 'disabled']

  let { win }: AppProps = $props()

  const ctx = getWindowContext()

  const saved = load('cron', { filter: 'mine' as Filter })
  let filter = $state<Filter>(FILTERS.includes(saved.filter) ? saved.filter : 'mine')
  let query = $state('')
  let table = $state.raw<CronTable | null>(null)
  let error = $state('')
  let loading = $state(false)
  let selected = $state<string | null>(null)
  /** The job dialog: a new job, an existing one, or closed. */
  let editing = $state.raw<CronJob | 'new' | null>(null)
  let pending = $state.raw<{ change: CronChange; plan: CronPlan } | null>(null)
  let busy = $state(false)
  let changeError = $state('')

  $effect(() => save('cron', { filter }))

  const server = $derived(servers.list.find((s) => s.id === win.serverID))
  const connected = $derived(server?.state === 'connected')
  const zone = $derived(table?.zone ?? 'UTC')

  function matches(j: CronJob, f: Filter): boolean {
    if (f === 'mine') return j.editable
    if (f === 'system') return !j.editable
    return !j.enabled
  }

  const shown = $derived.by(() => {
    const q = query.trim().toLowerCase()
    return (table?.jobs ?? []).filter(
      (j) =>
        matches(j, filter) &&
        (!q ||
          j.command.toLowerCase().includes(q) ||
          describe(j.schedule).toLowerCase().includes(q)),
    )
  })
  const current = $derived(table?.jobs.find((j) => j.id === selected) ?? null)

  const filters = $derived<FilterItem[]>(
    FILTERS.map((f) => ({
      id: f,
      label: t.cron.filters[f],
      icon: f === 'mine' ? 'clock' : f === 'system' ? 'server' : 'pause',
      count: table?.jobs.filter((j) => matches(j, f)).length,
    })),
  )

  const items = $derived<StatusItem[]>(
    shown.map((j) => ({
      key: j.id,
      name: j.command,
      health: j.enabled ? 'ok' : 'off',
      note: !j.enabled
        ? t.cron.off
        : j.next[0]
          ? formatRun(j.next[0], zone)
          : j.schedule.reboot
            ? t.cron.atBoot
            : t.cron.never,
      title: describe(j.schedule),
    })),
  )

  async function refresh() {
    if (loading) return
    loading = true
    try {
      table = await backend.list(win.serverID)
      error = ''
    } catch (e) {
      error = toUserError(e).message
    } finally {
      loading = false
    }
  }

  async function plan(change: CronChange) {
    busy = true
    changeError = ''
    try {
      pending = { change, plan: await backend.plan(win.serverID, change) }
      editing = null
    } catch (e) {
      changeError = toUserError(e).message
      // A job that changed under us: show what is there now.
      if (toUserError(e).code === 'FS_STALE') void refresh()
    } finally {
      busy = false
    }
  }

  function start(op: Exclude<CronOp, 'add'>, job: CronJob) {
    if (op === 'edit') editing = job
    else void plan({ op, id: job.id, schedule: '', command: '' })
  }

  async function confirm() {
    if (!pending) return
    busy = true
    changeError = ''
    try {
      await backend.apply(win.serverID, pending.change, pending.plan)
      if (pending.change.op === 'delete') selected = null
      pending = null
      await refresh()
    } catch (e) {
      const ue = toUserError(e)
      changeError = ue.code === 'FS_STALE' ? t.cron.stale : ue.message
      if (ue.code === 'FS_STALE') void refresh()
    } finally {
      busy = false
    }
  }

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
  bind:filter={() => filter, (f) => (filter = f as Filter)}
  sidebarLabel={t.cron.sections}
  searchLabel={t.cron.search}
  bind:query
  refreshing={!connected || loading}
  ready={connected && table !== null}
  footer={table && table.unreadable.length > 0
    ? `${t.cron.count(shown.length)} · ${t.cron.unreadable(table.unreadable.length)}`
    : t.cron.count(shown.length)}
  {error}
  onrefresh={() => void refresh()}
>
  {#snippet actions()}
    <button
      class="tool"
      aria-label={t.cron.add}
      title={t.cron.add}
      disabled={!connected || !table}
      onclick={() => {
        changeError = ''
        editing = 'new'
      }}
    >
      <Icon name="plus" size={15} strokeWidth={2.2} />
    </button>
  {/snippet}
  {#snippet placeholder()}
    {#if !connected}
      <Empty icon="server" title={t.cron.offline} hint={t.cron.offlineHint} />
    {:else}
      <Empty icon={error ? 'warning' : 'clock'} title={error || t.cron.loading} />
    {/if}
  {/snippet}
  {#snippet list()}
    <StatusList
      {items}
      {selected}
      label={t.cron.search}
      empty={filter === 'mine' && !query ? t.cron.noJobs : t.cron.noMatch}
      onselect={(id) => (selected = id)}
    />
  {/snippet}
  {#snippet detail()}
    {#if current}
      <CronInspector serverID={win.serverID} job={current} {zone} onchange={start} />
    {:else}
      <Empty icon="clock" title={t.cron.noSelection} />
    {/if}
  {/snippet}
</FilterShell>

<JobDialog
  job={editing}
  {zone}
  {busy}
  error={changeError}
  onclose={() => (editing = null)}
  onsubmit={(c) => void plan(c)}
/>
<ChangeDialog
  {pending}
  {busy}
  error={changeError}
  onclose={() => (pending = null)}
  onconfirm={() => void confirm()}
/>
