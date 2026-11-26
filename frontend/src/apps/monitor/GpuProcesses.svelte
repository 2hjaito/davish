<script lang="ts">
  import { toUserError } from '../../lib/api/errors'
  import { type GPUProcess, type GPUProcessList, monitorGPUProcesses } from '../../lib/api/monitor'
  import { hasRuntime } from '../../lib/api/runtime'
  import { formatBytes, formatPct } from '../../lib/format'
  import { t } from '../../lib/i18n'
  import Icon from '../../lib/ui/Icon.svelte'
  import Card from './Card.svelte'
  import {
    DEFAULT_GPU_ORDER,
    type GpuProcColumn,
    type GpuProcOrder,
    nextGpuOrder,
    oneLine,
    orderGpuProcs,
  } from './gpuProcSort'
  import { mockGpuProcesses } from './mock'

  interface Props {
    serverID: string
    paused: boolean
    /** With one GPU every row would say 0, so the GPU column is left out. */
    gpuCount: number
  }

  let { serverID, paused, gpuCount }: Props = $props()

  // Same pace as the Processes tab; nvidia-smi pmon itself samples for a second.
  const POLL_MS = 3000

  let list = $state.raw<GPUProcessList | null>(null)
  let error = $state('')
  let order = $state<GpuProcOrder>(DEFAULT_GPU_ORDER)

  const rows = $derived(list ? orderGpuProcs(list.procs, order) : [])

  $effect(() => {
    if (paused) return
    const id = serverID
    let stopped = false
    let timer: ReturnType<typeof setTimeout> | undefined
    // One request at a time: the next waits for the previous to answer.
    const poll = async () => {
      try {
        const next = hasRuntime ? await monitorGPUProcesses(id) : mockGpuProcesses()
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

  // opt columns give way to the command in a narrow window.
  const allColumns: { key: GpuProcColumn; label: string; num: boolean; opt: boolean }[] = [
    { key: 'pid', label: t.monitor.pid, num: true, opt: false },
    { key: 'user', label: t.monitor.procUser, num: false, opt: false },
    { key: 'gpu', label: t.monitor.gpu, num: true, opt: false },
    { key: 'type', label: t.monitor.gpuType, num: false, opt: true },
    { key: 'gpuUtil', label: t.monitor.gpuUtil, num: true, opt: false },
    { key: 'gpuMem', label: t.monitor.vram, num: true, opt: false },
    { key: 'cpu', label: t.monitor.procCpu, num: true, opt: true },
    { key: 'mem', label: t.monitor.hostMem, num: true, opt: true },
    { key: 'command', label: t.monitor.procCommand, num: false, opt: false },
  ]
  const multiGpu = $derived(gpuCount > 1)
  const columns = $derived(allColumns.filter((c) => c.key !== 'gpu' || multiGpu))

  const types: Record<string, string> = t.monitor.gpuTypes
  const known = (v: number, show: (v: number) => string) => (v >= 0 ? show(v) : t.monitor.unknown)
  const command = (p: GPUProcess) => oneLine(p.command)

  function ariaSort(key: GpuProcColumn) {
    if (order.column !== key) return 'none'
    return order.desc ? 'descending' : 'ascending'
  }
</script>

<Card title={t.monitor.processes} icon="list">
  {#if error && !list}
    <p class="note warn"><Icon name="warning" size={13} /> {error}</p>
  {:else if !list}
    <p class="note">{t.monitor.gpuProcsLoading}</p>
  {:else}
    {#if error}<p class="note warn"><Icon name="warning" size={13} /> {error}</p>{/if}
    <div class="scroll">
      <table class="mon-table mon-gpuprocs">
        <thead>
          <tr>
            {#each columns as c (c.key)}
              <th
                class:num={c.num}
                class:opt={c.opt}
                class:wide={c.key === 'command'}
                aria-sort={ariaSort(c.key)}
              >
                <button
                  class="sort"
                  class:active={order.column === c.key}
                  aria-label={t.monitor.sortBy(c.label)}
                  onclick={() => (order = nextGpuOrder(order, c.key))}
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
          {#each rows as p (`${p.gpu}:${p.pid}`)}
            <tr>
              <td class="num muted">{p.pid}</td>
              <td>{p.user || t.monitor.unknown}</td>
              {#if multiGpu}<td class="num muted">{p.gpu}</td>{/if}
              <td class="opt" class:graphics={p.type !== 'C'}>{types[p.type] ?? p.type}</td>
              <td class="num">{known(p.gpuUtil, formatPct)}</td>
              <td class="num">
                {known(p.gpuMem, formatBytes)}
                {#if p.gpuMemPct >= 0}<span class="muted pct">{formatPct(p.gpuMemPct)}</span>{/if}
              </td>
              <td class="num opt">{known(p.cpu, formatPct)}</td>
              <td class="num opt">{known(p.mem, formatBytes)}</td>
              <td class="cmd" title={command(p)}>
                <span class="name">{p.name}</span>
                {#if command(p) !== p.name}<span class="muted">{command(p)}</span>{/if}
              </td>
            </tr>
          {:else}
            <tr><td class="none" colspan={columns.length}>{t.monitor.gpuProcsNone}</td></tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</Card>

<style>
  .note {
    display: flex;
    align-items: center;
    gap: 6px;
    margin: 0;
    font-size: var(--text-sm);
    color: var(--fg-muted);
  }

  .note.warn {
    font-size: var(--text-xs);
    color: var(--warn);
  }

  .scroll {
    overflow-x: auto;
    margin: 0 -6px;
  }

  th {
    padding: 0;
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

  .pct {
    display: inline-block;
    min-width: 4ch;
    margin-left: 6px;
  }

  .graphics {
    color: var(--chart-2);
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

  @container window (max-width: 1000px) {
    .opt {
      display: none;
    }
  }
</style>
