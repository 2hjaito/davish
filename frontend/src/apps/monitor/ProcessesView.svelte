<script lang="ts">
  import { toUserError } from '../../lib/api/errors'
  import { monitorProcesses, type ProcessList } from '../../lib/api/monitor'
  import { hasRuntime } from '../../lib/api/runtime'
  import { formatBytes, formatDuration, formatPct } from '../../lib/format'
  import { t } from '../../lib/i18n/vi'
  import Icon from '../../lib/ui/Icon.svelte'
  import Empty from './Empty.svelte'
  import { mockProcesses } from './mock'
  import { nextOrder, orderProcs, type ProcColumn, type ProcOrder, serverSort } from './procSort'

  interface Props {
    serverID: string
    paused: boolean
  }

  let { serverID, paused }: Props = $props()

  // Often enough to follow a spike, rare enough that ps stays cheap.
  const POLL_MS = 3000

  let list = $state.raw<ProcessList | null>(null)
  let error = $state('')
  let order = $state<ProcOrder>({ column: 'cpu', desc: true })
  let query = $state('')

  const sortBy = $derived(serverSort(order.column))
  const rows = $derived(list ? orderProcs(list.procs, order, query) : [])

  $effect(() => {
    if (paused) return
    const id = serverID
    const by = sortBy
    let stopped = false
    let timer: ReturnType<typeof setTimeout> | undefined
    // One request at a time: the next waits for the previous to answer.
    const poll = async () => {
      try {
        const next = hasRuntime ? await monitorProcesses(id, by) : mockProcesses(by)
        if (stopped) return
        list = next
        error = ''
      } catch (e) {
        if (stopped) return
        error = toUserError(e).message
      }
      timer = setTimeout(() => void poll(), POLL_MS)
    }
    void poll()
    return () => {
      stopped = true
      clearTimeout(timer)
    }
  })

  const columns: { key: ProcColumn; label: string; num: boolean }[] = [
    { key: 'pid', label: t.monitor.pid, num: true },
    { key: 'name', label: t.monitor.procName, num: false },
    { key: 'user', label: t.monitor.procUser, num: false },
    { key: 'cpu', label: t.monitor.procCpu, num: true },
    { key: 'mem', label: t.monitor.procMem, num: true },
    { key: 'threads', label: t.monitor.threads, num: true },
    { key: 'age', label: t.monitor.age, num: true },
  ]

  function ariaSort(key: ProcColumn) {
    if (order.column !== key) return 'none'
    return order.desc ? 'descending' : 'ascending'
  }
</script>

<div class="procs">
  <div class="bar">
    <label class="search">
      <Icon name="search" size={13} strokeWidth={2.2} />
      <input
        type="text"
        placeholder={t.monitor.procSearch}
        aria-label={t.monitor.procSearch}
        bind:value={query}
      />
    </label>
    {#if list}
      <span class="count" title={t.monitor.procTop}>
        {t.monitor.procCount(rows.length, list.total)}
      </span>
    {/if}
  </div>

  {#if error && !list}
    <Empty icon="warning" title={error} />
  {:else if !list}
    <Empty icon="list" title={t.monitor.procLoading} />
  {:else}
    {#if error}<p class="error"><Icon name="warning" size={13} /> {error}</p>{/if}
    <div class="scroll">
      <table class="mon-table">
        <thead>
          <tr>
            {#each columns as c (c.key)}
              <th class:num={c.num} class:wide={c.key === 'name'} aria-sort={ariaSort(c.key)}>
                <button
                  class="sort"
                  class:active={order.column === c.key}
                  aria-label={t.monitor.sortBy(c.label)}
                  onclick={() => (order = nextOrder(order, c.key))}
                >
                  {c.label}
                  {#if order.column === c.key}
                    <Icon name={order.desc ? 'arrowDown' : 'arrowUp'} size={11} strokeWidth={2.4} />
                  {/if}
                </button>
              </th>
            {/each}
          </tr>
        </thead>
        <tbody>
          {#each rows as p (p.pid)}
            <tr>
              <td class="num muted">{p.pid}</td>
              <td class="cmd" title={p.command}>
                <span class="name">{p.name}</span>
                {#if p.command !== p.name}<span class="muted">{p.command}</span>{/if}
              </td>
              <td>{p.user}</td>
              <td class="num">{formatPct(p.cpu)}</td>
              <td class="num">{formatBytes(p.mem)}</td>
              <td class="num muted">{p.threads}</td>
              <td class="num muted">{formatDuration(p.age, t.units)}</td>
            </tr>
          {:else}
            <tr><td class="none" colspan={columns.length}>{t.monitor.procNoMatch}</td></tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>

<style>
  .procs {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }

  .search {
    display: flex;
    align-items: center;
    gap: 6px;
    background: var(--fill-soft);
    border-radius: 8px;
    padding: 5px 9px;
    font-size: var(--text-sm);
    color: var(--fg-muted);
  }

  .search input {
    border: 0;
    background: transparent;
    width: 190px;
    outline: none;
    color: var(--fg);
  }

  .count {
    font-size: var(--text-xs);
    color: var(--fg-muted);
  }

  .error {
    display: flex;
    align-items: center;
    gap: 6px;
    margin: 0;
    font-size: var(--text-xs);
    color: var(--warn);
  }

  .scroll {
    flex: 1;
    min-height: 0;
    overflow: auto;
    margin: 0 -6px;
  }

  thead th {
    position: sticky;
    top: 0;
    z-index: 1;
    padding: 0;
    background: var(--content-bg);
  }

  .sort {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    width: 100%;
    padding: 5px 10px;
    border: 0;
    background: none;
    font: inherit;
    color: inherit;
  }

  .num .sort {
    justify-content: flex-end;
  }

  .sort.active {
    color: var(--fg);
  }

  .sort:hover {
    color: var(--accent);
  }

  /* The command takes what the other columns leave and ends in an ellipsis. */
  .wide,
  .cmd {
    width: 100%;
  }

  .cmd {
    max-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .name {
    margin-right: 8px;
    font-weight: 500;
  }

  .none {
    text-align: center;
    color: var(--fg-subtle);
  }
</style>
