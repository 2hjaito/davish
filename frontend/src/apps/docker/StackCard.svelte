<script lang="ts">
  import type { Container, ContainerAction, ContainerStats } from '../../lib/api/docker'
  import { formatBytes, formatPct } from '../../lib/format'
  import { t } from '../../lib/i18n'
  import ActionMenu, { type Action } from '../../lib/ui/ActionMenu.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import { openInBrowser } from '../../lib/wm/intents.svelte'
  import { techIcons } from '../../lib/ui/techIcons'
  import { ago, canDo, type Group, health, stack } from './containers'
  import type { StackIcon } from '../../lib/api/docker'
  import { knownTech, projectIcon } from './projectIcon'
  import { groupTech } from './tech'

  interface Props {
    serverID: string
    group: Group
    stats: Record<string, ContainerStats>
    /** The compose project's container list is showing. */
    open: boolean
    busy: string | null
    /** The project's menu, or the lone container's. */
    actions: (Action | null)[]
    /** A container's own menu, for the rows of an open project. */
    menu: (c: Container) => (Action | null)[]
    ontoggle: () => void
    onopen: (name: string) => void
    onact: (c: Container, action: ContainerAction) => void
  }

  let { serverID, group, stats, open, busy, actions, menu, ontoggle, onopen, onact }: Props =
    $props()

  const d = t.docker
  const tb = t.docker.table

  const s = $derived(stack(group))
  const dir = $derived(group.project ? group.containers[0]?.projectDir : undefined)
  let found = $state.raw<StackIcon>({})
  $effect(() => {
    const d = dir
    found = {}
    if (!d) return
    let live = true
    void projectIcon(serverID, d).then((r) => {
      if (live) found = r
    })
    return () => {
      live = false
    }
  })
  // The project's own code says more than the images it runs.
  const tech = $derived(knownTech(found.tech) ?? groupTech(group))
  const lone = $derived(group.project ? undefined : group.containers[0])
  const usage = $derived.by(() => {
    const ss = group.containers.map((c) => stats[c.id]).filter((x) => x !== undefined)
    return ss.length
      ? { cpu: ss.reduce((a, x) => a + x.cpu, 0), mem: ss.reduce((a, x) => a + x.memUsed, 0) }
      : undefined
  })
  const running = (c: Container) =>
    c.state === 'running' || c.state === 'restarting' || c.state === 'paused'

  function activate() {
    if (lone) onopen(lone.name)
    else ontoggle()
  }
</script>

{#snippet power(c: Container)}
  {#if running(c)}
    <button
      class="icon-btn"
      aria-label={d.actions.stop}
      disabled={busy === c.name}
      onclick={() => onact(c, 'stop')}><Icon name="stop" size={13} /></button
    >
  {:else}
    <button
      class="icon-btn go"
      aria-label={d.actions.start}
      disabled={busy === c.name || !canDo(c.state, 'start')}
      onclick={() => onact(c, 'start')}><Icon name="play" size={14} /></button
    >
  {/if}
{/snippet}

<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions, a11y_no_noninteractive_element_interactions -->
<article class="sc" class:failed={s.health === 'failed'} onclick={activate}>
  <header>
    {#if found.icon}
      <span class="logo"><img src={found.icon} alt="" /></span>
    {:else if tech}
      {@const logo = techIcons[tech]}
      <span class="logo" style:color={logo.color ?? 'var(--fg)'} title={tech}>
        <Icon def={logo} size={20} />
      </span>
    {/if}
    <div class="title">
      <button
        class="name"
        aria-expanded={lone ? undefined : open}
        title={lone ? tb.details : open ? tb.fold : tb.unfold}
        onclick={(e) => (e.stopPropagation(), activate())}>{s.name}</button
      >
      <small>
        <span class="dk-dot {s.health}"></span>
        <span class="line"
          >{lone
            ? (d.states[lone.state] ?? lone.state)
            : tb.compose(s.up, s.total)}{#if s.age}{` · ${ago(s.age)}`}{/if}</span
        >
      </small>
    </div>
    <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
    <div class="tools" onclick={(e) => e.stopPropagation()}>
      {#if lone}{@render power(lone)}{/if}
      <ActionMenu label={lone ? tb.more(s.name) : d.compose.more(s.name)} {actions} />
    </div>
  </header>

  <p class="image" title={s.images.join('\n')}>
    <span class="mono">{s.images[0] ?? ''}</span>
    {#if s.images.length > 1}<span class="more">{tb.moreImages(s.images.length - 1)}</span>{/if}
  </p>

  <div class="ports">
    {#each s.ports as p (p.host)}
      <button
        class="dk-chip"
        title={tb.openPort(p.host ?? 0)}
        onclick={(e) => (e.stopPropagation(), openInBrowser(serverID, p.host ?? 0))}
        >{p.local ? '127.0.0.1:' : ':'}{p.host}</button
      >
    {:else}
      <span class="none">{tb.noPorts}</span>
    {/each}
  </div>

  {#if s.looping}
    <p class="warn"><Icon name="warning" size={13} />{tb.loop}</p>
  {/if}

  <footer>
    <span>CPU <b>{usage ? formatPct(usage.cpu) : '–'}</b></span>
    <span>RAM <b>{usage ? formatBytes(usage.mem) : '–'}</b></span>
    {#if !lone}
      <span class="chev" class:open aria-hidden="true"><Icon name="chevronRight" size={13} /></span>
    {/if}
  </footer>

  {#if open && !lone}
    <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_noninteractive_element_interactions -->
    <ul onclick={(e) => e.stopPropagation()}>
      {#each group.containers as c (c.id)}
        <li>
          <span class="dk-dot {health(c.state)}" title={d.states[c.state] ?? c.state}></span>
          <button class="row" title={tb.details} onclick={() => onopen(c.name)}>
            {c.service || c.name}
            <small>{d.states[c.state] ?? c.state}</small>
          </button>
          {@render power(c)}
          <ActionMenu label={tb.more(c.name)} actions={menu(c)} />
        </li>
      {/each}
    </ul>
  {/if}
</article>

<style>
  .sc {
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 12px 14px;
    min-width: 0;
    border-radius: var(--card-radius);
    background: var(--card-bg);
    border: 1px solid var(--divider-soft);
    cursor: pointer;
    transition: border-color 0.15s;
  }

  .sc:hover {
    border-color: var(--fill-track);
  }

  .sc.failed {
    border-color: color-mix(in srgb, var(--danger) 45%, transparent);
  }

  header {
    display: flex;
    align-items: center;
    gap: 10px;
    min-width: 0;
  }

  .logo {
    flex: none;
    display: grid;
    place-items: center;
    width: 34px;
    height: 34px;
    border-radius: 9px;
    background: var(--chart-surface);
    box-shadow: inset 0 0 0 1px var(--divider-soft);
  }

  .logo img {
    width: 22px;
    height: 22px;
    object-fit: contain;
  }

  .title {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .name {
    all: unset;
    font-weight: 600;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .name:focus-visible {
    outline: 2px solid var(--accent);
    border-radius: 4px;
  }

  .title small {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: var(--text-xs);
    color: var(--fg-muted);
  }

  .title .dk-dot {
    flex: none;
  }

  .line {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .tools {
    display: flex;
    align-items: center;
    gap: 2px;
  }

  .image {
    margin: 0;
    display: flex;
    gap: 6px;
    min-width: 0;
    font-size: var(--text-xs);
    color: var(--fg-muted);
  }

  .image .mono {
    font-family: var(--font-mono);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .image .more {
    flex: none;
  }

  .ports {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    min-height: 20px;
  }

  .ports .dk-chip {
    margin: 0;
  }

  .none {
    font-size: var(--text-xs);
    color: var(--fg-subtle);
  }

  .warn {
    margin: 0;
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: var(--text-xs);
    color: var(--danger);
  }

  footer {
    display: flex;
    align-items: center;
    gap: 14px;
    padding-top: 8px;
    border-top: 1px solid var(--divider-soft);
    font-size: var(--text-xs);
    color: var(--fg-muted);
    font-variant-numeric: tabular-nums;
  }

  footer b {
    font-weight: 600;
    color: var(--fg);
  }

  .chev {
    margin-left: auto;
    display: flex;
    transition: rotate 0.15s;
  }

  .chev.open {
    rotate: 90deg;
  }

  ul {
    margin: -2px 0 0;
    padding: 0;
    list-style: none;
    display: flex;
    flex-direction: column;
    gap: 2px;
    cursor: default;
  }

  li {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 2px 0 2px 2px;
  }

  .row {
    all: unset;
    flex: 1;
    min-width: 0;
    display: flex;
    align-items: baseline;
    gap: 8px;
    cursor: pointer;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
  }

  .row:hover {
    color: var(--accent);
  }

  .row:focus-visible {
    outline: 2px solid var(--accent);
    border-radius: 4px;
  }

  .row small {
    font-size: var(--text-xs);
    color: var(--fg-muted);
  }
</style>
