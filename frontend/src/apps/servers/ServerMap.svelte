<script lang="ts">
  import type { Container } from '../../lib/api/docker'
  import type { Mount } from '../../lib/api/disk'
  import { toUserError } from '../../lib/api/errors'
  import type { NginxSite } from '../../lib/api/nginx'
  import type { Port, PortList } from '../../lib/api/ports'
  import { formatBytes, formatPct } from '../../lib/format'
  import { t } from '../../lib/i18n'
  import Meter from '../../lib/ui/charts/Meter.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import { openDisk, openInBrowser, openSelected } from '../../lib/wm/intents.svelte'
  import { wm } from '../../lib/wm/windowManager.svelte'
  import { backend as diskBackend } from '../disk/backend'
  import { backend as dockerBackend } from '../docker/backend'
  import { groupHealth, groups } from '../docker/containers'
  import { backend as nginxBackend } from '../nginx/backend'
  import { backend as portsBackend } from '../ports/backend'
  import { apps } from '../registry'
  import type { AppId } from '../types'
  import { processes, type Route, routes, sortRoutes } from './map'

  interface Props {
    serverID: string
    name: string
    onback: () => void
  }

  let { serverID, name, onback }: Props = $props()

  const m = t.servers.map

  type Source = keyof typeof m.sources
  let loading = $state(true)
  let sites = $state.raw<NginxSite[]>([])
  let ports = $state.raw<Port[]>([])
  let containers = $state.raw<Container[]>([])
  let mounts = $state.raw<Mount[]>([])
  let failed = $state.raw<{ source: Source; why: string }[]>([])

  async function load() {
    loading = true
    const [n, p, d, k] = await Promise.allSettled([
      nginxBackend.list(serverID),
      portsBackend.list(serverID),
      dockerBackend.list(serverID),
      diskBackend.mounts(serverID),
    ])
    const bad: { source: Source; why: string }[] = []
    // A missing nginx or docker is a normal server, not a failure.
    const take = <T,>(r: PromiseSettledResult<T>, source: Source, empty: T): T => {
      if (r.status === 'fulfilled') return r.value
      const u = toUserError(r.reason)
      if (u.code !== 'CMD_MISSING') bad.push({ source, why: u.message })
      return empty
    }
    sites = take(n, 'nginx', [])
    ports = take<PortList | null>(p, 'ports', null)?.ports ?? []
    containers = take(d, 'docker', [])
    mounts = take(k, 'disk', [])
    failed = bad
    loading = false
  }

  $effect(() => {
    void serverID
    void load()
  })

  const list = $derived(
    sortRoutes(routes({ sites, ports, containers, now: Math.floor(Date.now() / 1000) })),
  )
  const stacks = $derived(groups(containers))
  const programs = $derived(processes(ports))

  function open(id: AppId) {
    const app = apps.find((a) => a.id === id)
    if (app) wm.openOrFocus(id, serverID, { title: app.label, size: app.size })
  }

  function openContainer(c: Container) {
    const app = apps.find((a) => a.id === 'docker')
    if (app) openSelected('docker', serverID, c.name, { title: app.label, size: app.size })
  }

  const warnApp: Record<Route['warnings'][number], AppId> = {
    noListener: 'nginx',
    certExpiring: 'nginx',
    dbExposed: 'ports',
  }
</script>

{#snippet arrow()}
  <span class="arrow" aria-hidden="true"><Icon name="chevronRight" size={12} /></span>
{/snippet}

<div class="map">
  <header class="bar">
    <button class="back" onclick={onback}>
      <span style:rotate="180deg" style:display="flex"><Icon name="chevronRight" size={13} /></span>
      {m.back}
    </button>
    <h2>{m.title(name)}</h2>
    <button class="icon-btn" aria-label={t.common.refresh} title={t.common.refresh} onclick={load}>
      <Icon name="refresh" size={14} />
    </button>
  </header>

  {#if loading}
    <p class="muted pad">{m.loading}</p>
  {:else}
    {#each failed as f (f.source)}
      <p class="unread">
        <Icon name="warning" size={12} />{m.unreadable(m.sources[f.source], f.why)}
      </p>
    {/each}

    <section>
      <h3>{m.exposed}</h3>
      {#if list.length === 0}
        <p class="muted">{m.noRoutes}</p>
      {:else}
        <ul class="routes">
          {#each list as r (r.key)}
            <li class:warn={r.warnings.length > 0}>
              <div class="chain">
                {#if r.domain || r.site}
                  <span class="chip domain" title={r.site}>
                    {#if r.tls}<Icon name="lock" size={11} />{/if}
                    {r.domain ?? r.site}
                    {#if r.aliases}<small>{m.aliases(r.aliases)}</small>{/if}
                  </span>
                  {@render arrow()}
                  <button class="chip" title={m.openIn(t.apps.nginx)} onclick={() => open('nginx')}>
                    nginx{#if r.listen.length}<small>:{r.listen.join(', :')}</small>{/if}
                  </button>
                {:else}
                  <span class="chip muted-chip">{m.direct}</span>
                {/if}

                {#if r.root}
                  {@render arrow()}
                  <span class="chip path" title={r.root}>{m.static} {r.root}</span>
                {:else if r.target?.kind === 'socket'}
                  {@render arrow()}
                  <span class="chip path" title={r.target.path}>{m.socket} {r.target.path}</span>
                {:else if r.target?.kind === 'name' && !r.container}
                  {@render arrow()}
                  <span class="chip">{m.upstream} {r.target.name}</span>
                {:else if r.target?.kind === 'port' || r.listener}
                  {@const p = r.listener?.port ?? (r.target?.kind === 'port' ? r.target.port : 0)}
                  {#if r.domain || r.site}{@render arrow()}{/if}
                  <button
                    class="chip port"
                    title={t.docker.table.openPort(p)}
                    onclick={() => openInBrowser(serverID, p)}>:{p}</button
                  >
                {/if}

                {#if r.container}
                  {@render arrow()}
                  <button
                    class="chip box"
                    title={m.openIn(t.apps.docker)}
                    onclick={() => r.container && openContainer(r.container)}
                  >
                    <Icon name="docker" size={11} />{r.container.service ?? r.container.name}
                    {#if r.container.project}<small>{r.container.project}</small>{/if}
                  </button>
                {:else if r.listener?.process}
                  {@render arrow()}
                  <span class="chip">{r.listener.process}</span>
                {/if}
              </div>
              {#each r.warnings as w (w)}
                <button class="warning" onclick={() => open(warnApp[w])}>
                  <Icon name="warning" size={12} />{m.warnings[w]}
                  <span class="go"
                    >{m.openIn(apps.find((a) => a.id === warnApp[w])?.label ?? '')}</span
                  >
                </button>
              {/each}
            </li>
          {/each}
        </ul>
      {/if}
    </section>

    <section>
      <h3>{m.running}</h3>
      {#if stacks.length === 0 && programs.length === 0}
        <p class="muted">{m.noStacks}</p>
      {:else}
        <div class="tags">
          {#each stacks as g (g.key)}
            {@const first = g.containers[0]}
            <button class="tag" onclick={() => first && openContainer(first)}>
              <span class="dot {groupHealth(g)}"></span>
              {g.project ?? first?.name}
              {#if g.project}<small>{g.containers.length}</small>{/if}
            </button>
          {/each}
          {#each programs as p (p)}
            <span class="tag plain"><Icon name="services" size={11} />{p}</span>
          {/each}
        </div>
      {/if}
    </section>

    {#if mounts.length}
      <section>
        <h3>{m.storage}</h3>
        <ul class="mounts">
          {#each mounts as k (k.path)}
            {@const pct = k.total ? (k.used / k.total) * 100 : 0}
            <li>
              <button class="mount" onclick={() => openDisk(serverID, k.path)}>
                <span class="mono">{k.path}</span>
                <Meter {pct} label={k.path} />
                <span class="val"
                  >{formatBytes(k.used)} / {formatBytes(k.total)} · {formatPct(pct)}</span
                >
              </button>
            </li>
          {/each}
        </ul>
      </section>
    {/if}

    <p class="muted hint">{m.hint}</p>
  {/if}
</div>

<style>
  .map {
    display: flex;
    flex-direction: column;
    gap: 14px;
    min-width: 0;
  }

  .bar {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .bar h2 {
    flex: 1;
    margin: 0;
    font-size: var(--text-md);
    font-weight: 600;
    text-align: center;
  }

  .back {
    display: flex;
    align-items: center;
    gap: 4px;
    border: 0;
    background: none;
    color: var(--accent);
    font: inherit;
  }

  section {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  h3 {
    margin: 0;
    font-size: var(--text-xs);
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--fg-muted);
  }

  .muted {
    margin: 0;
    color: var(--fg-muted);
    font-size: var(--text-sm);
  }

  .pad {
    padding: 24px 0;
    text-align: center;
  }

  .hint {
    font-size: var(--text-xs);
  }

  .unread {
    margin: 0;
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: var(--text-xs);
    color: var(--warn);
  }

  .routes,
  .mounts {
    margin: 0;
    padding: 0;
    list-style: none;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .routes li {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 8px 10px;
    border-radius: var(--card-radius);
    background: var(--card-bg);
    border: 1px solid var(--divider-soft);
  }

  .routes li.warn {
    border-color: color-mix(in srgb, var(--danger) 40%, transparent);
  }

  .chain {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 4px;
    min-width: 0;
  }

  .chip,
  .tag {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    max-width: 100%;
    padding: 2px 8px;
    border: 0;
    border-radius: 10px;
    font: inherit;
    font-size: var(--text-xs);
    background: var(--fill-soft);
    color: var(--fg);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  button.chip,
  button.tag {
    cursor: pointer;
  }

  button.chip:hover,
  button.tag:hover {
    background: var(--accent-soft);
    color: var(--accent);
  }

  .chip small,
  .tag small {
    color: var(--fg-muted);
  }

  .chip.domain {
    font-weight: 600;
  }

  .chip.port {
    font-variant-numeric: tabular-nums;
    background: var(--accent-soft);
    color: var(--accent);
  }

  .chip.path {
    font-family: var(--font-mono);
  }

  .muted-chip {
    color: var(--fg-muted);
  }

  .arrow {
    display: flex;
    color: var(--fg-subtle);
  }

  .warning {
    display: flex;
    align-items: center;
    gap: 6px;
    border: 0;
    padding: 0;
    background: none;
    font: inherit;
    font-size: var(--text-xs);
    color: var(--danger);
    text-align: left;
  }

  .warning .go {
    color: var(--accent);
  }

  .tags {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    border: 1.5px solid var(--fg-subtle);
    box-sizing: border-box;
  }

  .dot.ok {
    background: var(--ok);
    border-color: var(--ok);
  }

  .dot.busy {
    background: var(--warn);
    border-color: var(--warn);
  }

  .dot.failed {
    background: var(--danger);
    border-color: var(--danger);
  }

  .tag.plain {
    color: var(--fg-muted);
  }

  .mount {
    display: grid;
    grid-template-columns: minmax(80px, 160px) 1fr auto;
    align-items: center;
    gap: 10px;
    width: 100%;
    padding: 6px 10px;
    border: 0;
    border-radius: 8px;
    background: none;
    font: inherit;
    font-size: var(--text-sm);
    text-align: left;
  }

  .mount:hover {
    background: var(--fill-soft);
  }

  .mono {
    font-family: var(--font-mono);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .val {
    font-size: var(--text-xs);
    color: var(--fg-muted);
    font-variant-numeric: tabular-nums;
  }
</style>
