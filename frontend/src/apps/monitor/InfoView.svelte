<script lang="ts">
  import { toUserError } from '../../lib/api/errors'
  import { monitorSystemInfo, type SystemInfo } from '../../lib/api/monitor'
  import { hasRuntime } from '../../lib/api/runtime'
  import { formatBytes, formatDuration } from '../../lib/format'
  import { locale, t } from '../../lib/i18n'
  import type { Sample } from '../../lib/stores/metrics.svelte'
  import Empty from '../../lib/ui/Empty.svelte'
  import { openDisk } from '../../lib/wm/intents.svelte'
  import Card from './Card.svelte'
  import { mockInfo } from './mock'

  interface Props {
    serverID: string
    /** Live figures (cores, memory, uptime) the info script does not read. */
    latest: Sample
  }

  let { serverID, latest }: Props = $props()

  const m = t.monitor
  const si = t.monitor.sysinfo

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

  const or = (v: string | undefined) => v || m.unknown

  const booted = $derived(
    info?.bootTime
      ? new Date(info.bootTime * 1000).toLocaleString(locale, {
          day: 'numeric',
          month: 'short',
          year: 'numeric',
          hour: '2-digit',
          minute: '2-digit',
        })
      : '',
  )
  const threads = $derived(info?.threads || latest.cores.length)
  const memTotal = $derived(info?.memTotal || latest.memTotal)
  const swapTotal = $derived(info?.swapTotal ?? latest.swapTotal)

  type Row = [string, string]
  const systemRows = $derived.by((): Row[] => {
    if (!info) return []
    const up = formatDuration(latest.uptime, t.units)
    const out: Row[] = [
      [m.hostname, or(info.hostname)],
      [si.osKernel, or(info.os)],
      [m.kernel, `${or(info.kernel)} · ${or(info.arch)}`],
      [m.virt, info.virt || m.bareMetal],
      [si.booted, booted ? `${booted} · ${up}` : up],
    ]
    if (info.timezone) out.push([si.timezone, info.timezone])
    if (info.systemd) out.push(['systemd', info.systemd])
    out.push([si.users, info.users?.length ? info.users.join(', ') : si.noUsers])
    return out
  })
  const hardwareRows = $derived.by((): Row[] =>
    info
      ? [
          [si.cpu, or(info.cpuModel)],
          [' ', si.layout(info.cores ?? 0, threads, info.sockets ?? 0)],
          [m.memory, formatBytes(memTotal)],
          [m.swap, swapTotal > 0 ? formatBytes(swapTotal) : m.unknown],
        ]
      : [],
  )
</script>

{#snippet rows(list: Row[])}
  <dl>
    {#each list as [label, value] (label)}
      <dt>{label}</dt>
      <dd>{value}</dd>
    {/each}
  </dl>
{/snippet}

{#if error}
  <Empty icon="warning" title={error} />
{:else if !info}
  <Empty icon="info" title={m.infoLoading} />
{:else}
  <div class="grid">
    <Card title={si.system} icon="server">
      {@render rows(systemRows)}
    </Card>

    <Card title={si.hardware} icon="cpu">
      {@render rows(hardwareRows)}
    </Card>

    <Card title={si.disks} icon="disk">
      {#if info.disks?.length}
        <ul class="disks">
          {#each info.disks as d (d.name)}
            <li>
              <div class="disk">
                {#if d.kind}<span class="kind {d.kind}">{si.kinds[d.kind]}</span>{/if}
                <strong>{d.model || d.name}</strong>
                <span class="muted">{d.model ? `${d.name} · ` : ''}{formatBytes(d.size)}</span>
              </div>
              {#each d.parts ?? [] as p (p.name)}
                <div class="part">
                  {#if p.mount}
                    {@const mount = p.mount}
                    <button
                      class="mount"
                      title={si.openDisk}
                      onclick={() => openDisk(serverID, mount)}>{mount}</button
                    >
                  {:else}
                    <span class="mono muted">{p.name}</span>
                  {/if}
                  <span class="muted">{p.fstype || si.noFs} · {formatBytes(p.size)}</span>
                </div>
              {/each}
            </li>
          {/each}
        </ul>
      {:else}
        <p class="muted">{si.noDisks}</p>
      {/if}
    </Card>

    <Card title={si.network} icon="network">
      {#if info.interfaces?.length}
        <ul class="ifaces">
          {#each info.interfaces as n (n.name)}
            <li>
              <span class="kind {n.kind}">{si.ifKinds[n.kind]}</span>
              <span class="mono name" class:off={!n.up}>{n.name}</span>
              <span class="addrs">
                {#each n.addrs ?? [] as a (a)}<span class="mono">{a}</span>{/each}
                {#if !n.up}<span class="muted">{si.down}</span>{/if}
              </span>
            </li>
          {/each}
        </ul>
      {:else}
        <p class="muted">{si.noNet}</p>
      {/if}
      {#if info.gateway || info.dns?.length}
        <dl class="net">
          {#if info.gateway}
            <dt>{si.gateway}</dt>
            <dd class="mono">{info.gateway}{info.gatewayDev ? ` · ${info.gatewayDev}` : ''}</dd>
          {/if}
          {#if info.dns?.length}
            <dt>{si.dns}</dt>
            <dd class="mono">{info.dns.join(', ')}</dd>
          {/if}
        </dl>
      {/if}
    </Card>
  </div>
{/if}

<style>
  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
    gap: 12px;
    align-items: start;
  }

  dl {
    display: grid;
    grid-template-columns: max-content 1fr;
    gap: 7px 18px;
    margin: 0;
    font-size: var(--text-sm);
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

  .muted {
    margin: 0;
    color: var(--fg-muted);
    font-size: var(--text-xs);
  }

  .mono {
    font-family: var(--font-mono);
    font-size: var(--text-xs);
  }

  ul {
    margin: 0;
    padding: 0;
    list-style: none;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .disk {
    display: flex;
    align-items: baseline;
    flex-wrap: wrap;
    gap: 6px;
    font-size: var(--text-sm);
  }

  .part {
    display: flex;
    align-items: baseline;
    gap: 8px;
    margin: 4px 0 0 14px;
    font-size: var(--text-xs);
  }

  .mount {
    border: 0;
    padding: 0;
    background: none;
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    color: var(--accent);
  }

  .mount:hover {
    text-decoration: underline;
  }

  .kind {
    flex: none;
    padding: 0 6px;
    border-radius: 6px;
    font-size: 10px;
    font-weight: 700;
    line-height: 17px;
    letter-spacing: 0.02em;
    background: var(--fill-soft);
    color: var(--fg-muted);
  }

  .kind.nvme,
  .kind.lan {
    background: var(--accent-soft);
    color: var(--accent);
  }

  .kind.hdd {
    background: color-mix(in srgb, var(--warn) 18%, transparent);
    color: var(--warn);
  }

  .kind.vpn {
    background: color-mix(in srgb, var(--ok) 18%, transparent);
    color: var(--ok);
  }

  .ifaces li {
    display: grid;
    grid-template-columns: 58px minmax(70px, max-content) 1fr;
    align-items: baseline;
    gap: 8px;
  }

  .ifaces .kind {
    justify-self: start;
  }

  .name.off {
    color: var(--fg-muted);
  }

  .addrs {
    display: flex;
    flex-wrap: wrap;
    gap: 2px 10px;
    min-width: 0;
  }

  .net {
    margin-top: 12px;
    padding-top: 10px;
    border-top: 1px solid var(--divider-soft);
  }
</style>
