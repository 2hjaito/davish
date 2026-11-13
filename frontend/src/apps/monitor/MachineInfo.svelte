<script lang="ts">
  import type { DiskUse } from '../../lib/api/docker'
  import { toUserError } from '../../lib/api/errors'
  import { monitorSystemInfo, type SystemInfo } from '../../lib/api/monitor'
  import { hasRuntime } from '../../lib/api/runtime'
  import { formatBytes, formatDuration } from '../../lib/format'
  import { locale, t } from '../../lib/i18n'
  import type { Sample } from '../../lib/stores/metrics.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import { openDisk } from '../../lib/wm/intents.svelte'
  import { backend as dockerBackend } from '../docker/backend'
  import Card from './Card.svelte'
  import { useByMount } from './dockerDisks'
  import { mockInfo } from './mock'

  interface Props {
    serverID: string
    /** Live figures (cores, memory, uptime) the info script does not read. */
    latest: Sample
  }

  let { serverID, latest }: Props = $props()

  const m = t.monitor
  const si = t.monitor.sysinfo

  let info = $state.raw<SystemInfo | null>(null)
  let error = $state('')
  /** Docker's data and the containers' mounts; null without docker or access. */
  let docker = $state.raw<DiskUse | null>(null)
  let showBridges = $state(false)

  $effect(() => {
    const id = serverID
    let stopped = false
    info = null
    error = ''
    docker = null
    const load = hasRuntime ? monitorSystemInfo(id) : Promise.resolve(mockInfo)
    load.then(
      (i) => {
        if (!stopped) info = i
      },
      (e: unknown) => {
        if (!stopped) error = toUserError(e).message
      },
    )
    // Optional: no docker, no access or a sudo password to ask just leaves it out.
    dockerBackend.diskUse(id).then(
      (u) => {
        if (!stopped) docker = u
      },
      () => {},
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

  type Row = [string, string]
  const systemRows = $derived.by((): Row[] => {
    if (!info) return []
    const up = formatDuration(latest.uptime, t.units)
    const out: Row[] = [
      [m.hostname, or(info.hostname)],
      [si.osKernel, or(info.os)],
      [m.kernel, or(info.kernel)],
      [si.booted, booted ? `${booted} · ${up}` : up],
    ]
    if (info.timezone) out.push([si.timezone, info.timezone])
    if (info.systemd) out.push(['systemd', info.systemd])
    out.push([si.users, info.users?.length ? info.users.join(', ') : si.noUsers])
    return out
  })
  const hardwareRows = $derived.by((): Row[] => {
    if (!info) return []
    const threads = info.threads || latest.cores.length
    const swap = info.swapTotal ?? latest.swapTotal
    return [
      [si.cpu, or(info.cpuModel)],
      [si.coresLabel, si.layout(info.cores ?? 0, threads, info.sockets ?? 0)],
      [m.memory, formatBytes(info.memTotal || latest.memTotal)],
      [m.swap, swap > 0 ? formatBytes(swap) : si.noSwap],
      [m.arch, or(info.arch)],
      [m.virt, info.virt || m.bareMetal],
    ]
  })

  const mountPoints = $derived(
    (info?.disks ?? []).flatMap((d) => (d.parts ?? []).flatMap((p) => (p.mount ? [p.mount] : []))),
  )
  const byMount = $derived(docker ? useByMount(docker, mountPoints) : new Map())

  // Docker bridges add a row per network; they fold into one.
  const hosts = $derived((info?.interfaces ?? []).filter((n) => n.kind !== 'docker'))
  const bridges = $derived((info?.interfaces ?? []).filter((n) => n.kind === 'docker'))
</script>

{#snippet rows(list: Row[])}
  <dl>
    {#each list as [label, value] (label)}
      <dt>{label}</dt>
      <dd>{value}</dd>
    {/each}
  </dl>
{/snippet}

{#snippet ifaceRow(n: NonNullable<SystemInfo['interfaces']>[number])}
  <tr class:off={!n.up}>
    <td class="mono">
      <span class="dot" class:up={n.up} title={n.up ? '' : si.down}></span>{n.name}
    </td>
    <td><span class="kind {n.kind}">{si.ifKinds[n.kind]}</span></td>
    <td class="mono addrs">{n.addrs?.join(', ') || '--'}</td>
  </tr>
{/snippet}

{#if error}
  <p class="error"><Icon name="warning" size={13} />{error}</p>
{:else if info}
  <div class="machine">
    <div class="pair">
      <Card title={si.system} icon="server">{@render rows(systemRows)}</Card>
      <Card title={si.hardware} icon="cpu">{@render rows(hardwareRows)}</Card>
    </div>

    <Card title={si.disks} icon="disk">
      {#if info.disks?.length}
        <table class="mon-table">
          <thead>
            <tr>
              <th>{si.colMount}</th>
              <th>{si.colFs}</th>
              <th class="num">{si.colSize}</th>
              {#if docker}<th>{si.colDocker}</th>{/if}
            </tr>
          </thead>
          <tbody>
            {#each info.disks as d (d.name)}
              <tr class="disk">
                <td colspan="2">
                  {#if d.kind}<span class="kind {d.kind}">{si.kinds[d.kind]}</span>{/if}
                  <strong>{d.model || d.name}</strong>
                  {#if d.model}<span class="muted">{d.name}</span>{/if}
                </td>
                <td class="num"><strong>{formatBytes(d.size)}</strong></td>
                {#if docker}<td></td>{/if}
              </tr>
              {#each d.parts ?? [] as p (p.name)}
                {@const use = p.mount ? byMount.get(p.mount) : undefined}
                <tr>
                  <td class="indent">
                    {#if p.mount}
                      {@const mount = p.mount}
                      <button
                        class="link mono"
                        title={si.openDisk}
                        onclick={() => openDisk(serverID, mount)}>{mount}</button
                      >
                    {:else}
                      <span class="mono muted">{p.name}</span>
                    {/if}
                  </td>
                  <td class="muted">{p.fstype || '--'}</td>
                  <td class="num">{formatBytes(p.size)}</td>
                  {#if docker}
                    <td class="owners" title={use?.owners.join(', ')}>
                      {#if use?.root}<span class="kind docker" title={docker.rootDir}>Docker</span
                        >{/if}{use?.owners.join(' · ') ?? ''}
                    </td>
                  {/if}
                </tr>
              {/each}
            {/each}
          </tbody>
        </table>
        {#if docker}
          <p class="foot muted">{si.dockerLine(docker.version, docker.rootDir)}</p>
        {/if}
      {:else}
        <p class="muted">{si.noDisks}</p>
      {/if}
    </Card>

    <Card title={si.network} icon="network">
      {#if hosts.length || bridges.length}
        <table class="mon-table">
          <thead>
            <tr>
              <th>{si.colIface}</th>
              <th>{si.colKind}</th>
              <th>{si.colAddr}</th>
            </tr>
          </thead>
          <tbody>
            {#each hosts as n (n.name)}{@render ifaceRow(n)}{/each}
            {#if bridges.length}
              <tr class="fold">
                <td colspan="3">
                  <button
                    class="link"
                    aria-expanded={showBridges}
                    onclick={() => (showBridges = !showBridges)}
                  >
                    <span class="chev" class:open={showBridges}
                      ><Icon name="chevronRight" size={12} /></span
                    >
                    {si.bridges(bridges.length)}
                    <span class="muted">· {showBridges ? si.hide : si.show}</span>
                  </button>
                </td>
              </tr>
              {#if showBridges}
                {#each bridges as n (n.name)}{@render ifaceRow(n)}{/each}
              {/if}
            {/if}
          </tbody>
        </table>
      {:else}
        <p class="muted">{si.noNet}</p>
      {/if}
      {#if info.gateway || info.dns?.length}
        <p class="foot muted">
          {#if info.gateway}{si.gateway}
            <span class="mono">{info.gateway}</span>{info.gatewayDev
              ? ` (${info.gatewayDev})`
              : ''}{/if}{#if info.gateway && info.dns?.length}&nbsp;·&nbsp;{/if}{#if info.dns?.length}{si.dns}
            <span class="mono">{info.dns.join(', ')}</span>{/if}
        </p>
      {/if}
    </Card>
  </div>
{/if}

<style>
  .machine {
    display: flex;
    flex-direction: column;
    gap: 12px;
    margin-top: 14px;
  }

  /* Two cards side by side, as tall as each other. */
  .pair {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    gap: 12px;
  }

  dl {
    display: grid;
    grid-template-columns: max-content 1fr;
    gap: 8px 20px;
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

  .error {
    display: flex;
    align-items: center;
    gap: 6px;
    margin: 14px 0 0;
    font-size: var(--text-sm);
    color: var(--warn);
  }

  .muted {
    color: var(--fg-muted);
  }

  p.muted {
    margin: 0;
    font-size: var(--text-sm);
  }

  .mono {
    font-family: var(--font-mono);
    font-size: var(--text-sm);
  }

  .num {
    text-align: right;
    font-variant-numeric: tabular-nums;
  }

  /* A drive heads its partitions: no stripe or hover, a rule above the next one. */
  :global(.mon-table) tbody tr.disk td {
    background: none;
  }

  tr.disk:not(:first-child) td {
    border-top: 1px solid var(--divider-soft);
  }

  tr.disk strong {
    font-weight: 600;
  }

  tr.disk .kind {
    margin-right: 8px;
  }

  tr.disk .muted {
    margin-left: 6px;
    font-size: var(--text-sm);
  }

  td.indent {
    padding-left: 22px;
  }

  td.owners {
    max-width: 320px;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  td.owners .kind {
    margin-right: 8px;
  }

  td.addrs {
    white-space: normal;
  }

  tr.off td {
    color: var(--fg-muted);
  }

  .dot {
    display: inline-block;
    width: 7px;
    height: 7px;
    margin-right: 8px;
    border-radius: 50%;
    background: var(--fg-subtle);
    vertical-align: middle;
  }

  .dot.up {
    background: var(--ok);
  }

  .link {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    border: 0;
    padding: 0;
    background: none;
    font: inherit;
    color: var(--accent);
  }

  .link.mono {
    font-family: var(--font-mono);
    font-size: var(--text-sm);
  }

  .link:hover {
    text-decoration: underline;
  }

  tr.fold .link {
    color: var(--fg);
  }

  tr.fold .link:hover {
    text-decoration: none;
  }

  .chev {
    display: flex;
    transition: rotate var(--dur-fast) var(--ease-out);
  }

  .chev.open {
    rotate: 90deg;
  }

  .foot {
    padding-top: 10px;
    border-top: 1px solid var(--divider-soft);
  }

  .kind {
    display: inline-block;
    padding: 0 6px;
    border-radius: 6px;
    font-size: var(--text-xs);
    font-weight: 600;
    line-height: 18px;
    background: var(--fill-soft);
    color: var(--fg-muted);
  }

  .kind.nvme,
  .kind.lan,
  .kind.docker {
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
</style>
