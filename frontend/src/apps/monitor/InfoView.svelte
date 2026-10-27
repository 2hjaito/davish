<script lang="ts">
  import { toUserError } from '../../lib/api/errors'
  import { monitorSystemInfo, type SystemInfo } from '../../lib/api/monitor'
  import { hasRuntime } from '../../lib/api/runtime'
  import { formatBytes, formatDuration } from '../../lib/format'
  import type { Sample } from '../../lib/stores/metrics.svelte'
  import { t } from '../../lib/i18n/vi'
  import Card from './Card.svelte'
  import Empty from './Empty.svelte'
  import { mockInfo } from './mock'

  interface Props {
    serverID: string
    /** Live figures (cores, memory, uptime) the info script does not read. */
    latest: Sample
  }

  let { serverID, latest }: Props = $props()

  let info = $state<SystemInfo | null>(null)
  let error = $state('')

  $effect(() => {
    const id = serverID
    let stopped = false
    info = null
    error = ''
    const load = hasRuntime ? monitorSystemInfo(id) : Promise.resolve(mockInfo)
    load.then(
      (i) => {
        if (!stopped) info = i
      },
      (e: unknown) => {
        if (!stopped) error = toUserError(e).message
      },
    )
    return () => {
      stopped = true
    }
  })

  const or = (v: string) => v || t.monitor.unknown
  const rows = $derived(
    info
      ? [
          [t.monitor.hostname, or(info.hostname)],
          [t.monitor.os, or(info.os)],
          [t.monitor.kernel, or(info.kernel)],
          [t.monitor.arch, or(info.arch)],
          [t.monitor.cpuModel, `${or(info.cpuModel)} · ${t.monitor.cores(latest.cores.length)}`],
          [t.monitor.memory, formatBytes(latest.memTotal)],
          [
            t.monitor.swap,
            latest.swapTotal > 0 ? formatBytes(latest.swapTotal) : t.monitor.unknown,
          ],
          [t.monitor.uptime, formatDuration(latest.uptime, t.units)],
          [t.monitor.virt, info.virt || t.monitor.bareMetal],
        ]
      : [],
  )
</script>

{#if error}
  <Empty icon="warning" title={error} />
{:else if !info}
  <Empty icon="info" title={t.monitor.infoLoading} />
{:else}
  <Card title={info.hostname || t.monitor.info} icon="server">
    <dl>
      {#each rows as [label, value] (label)}
        <dt>{label}</dt>
        <dd>{value}</dd>
      {/each}
    </dl>
  </Card>
{/if}

<style>
  dl {
    display: grid;
    grid-template-columns: max-content 1fr;
    gap: 9px 24px;
    margin: 0;
    font-size: var(--text-md);
  }

  dt {
    color: var(--fg-muted);
  }

  dd {
    margin: 0;
    min-width: 0;
    overflow-wrap: anywhere;
    font-weight: 500;
  }
</style>
