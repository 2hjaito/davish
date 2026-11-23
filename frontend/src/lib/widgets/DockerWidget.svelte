<script lang="ts">
  import { untrack } from 'svelte'

  import { hasRuntime } from '../api/runtime'
  import { t } from '../i18n'
  import { dockerHealth } from '../stores/dockerHealth.svelte'
  import { servers } from '../stores/servers.svelte'
  import Dots from './Dots.svelte'
  import { hasProblem, problemCount } from './dockerHealth'
  import { LOOK } from './look'
  import type { WidgetProps } from './registry'
  import WidgetBadge from './WidgetBadge.svelte'
  import WidgetCard, { type Tone } from './WidgetCard.svelte'

  let { serverID, size }: WidgetProps = $props()

  const d = t.widgets.docker

  const connected = $derived(
    !hasRuntime || servers.list.find((s) => s.id === serverID)?.state === 'connected',
  )

  // One `docker ps` a minute for the server, shared with every Docker widget
  // watching it (stores/dockerHealth.svelte.ts); refresh reads state of its
  // own, which must not become a dependency here.
  $effect(() => dockerHealth.watch(serverID))
  $effect(() => {
    if (connected) untrack(() => void dockerHealth.refresh(serverID))
  })

  const r = $derived(dockerHealth.of(serverID))
  const s = $derived(r.summary)

  const message = $derived.by(() => {
    if (!connected) return d.notConnected
    if (r.problem === 'missing') return d.missing
    if (r.problem === 'needsRoot') return d.needsRoot
    if (!s) return r.problem === 'failed' ? d.failed : t.widgets.waiting
    return ''
  })

  const bad = $derived(s ? hasProblem(s) : false)
  const tone = $derived<Tone>(
    !s || message ? 'neutral' : bad ? 'danger' : s.starting > 0 ? 'warn' : 'ok',
  )

  /** A short word for the state: how many problems, else what is starting, else that all is well. */
  const line = $derived.by(() => {
    if (!s) return ''
    if (s.total === 0) return d.none
    if (bad && s) return d.problems(problemCount(s))
    if (s.starting > 0) return d.starting(s.starting)
    return d.allHealthy
  })

  const COLORS = {
    ok: 'var(--ok-on-dark)',
    bad: 'var(--danger-on-dark)',
    starting: 'var(--warn-on-dark)',
    stopped: 'var(--widget-stopped)',
  }

  /** One row of dots in the small widget, a grid in the medium one. */
  const COLS = $derived(size === 'M' ? 12 : 14)
  const dots = $derived(s ? s.dots.slice(0, size === 'M' ? 60 : COLS) : [])

  /** The containers to name in the medium widget: the failing ones first. */
  const NAMED = 3
  const names = $derived(
    s
      ? [
          ...s.unhealthy.map((name) => ({ name, kind: 'unhealthy' as const })),
          ...s.restarting.map((name) => ({ name, kind: 'restarting' as const })),
        ]
      : [],
  )
</script>

<WidgetCard {tone} accent={LOOK.docker.accent}>
  <div class="docker">
    <div class="head">
      <span class="title"
        ><WidgetBadge kind="docker" /><span class="label">{t.apps.docker}</span></span
      >
      {#if s && !message}
        <span class="state {tone}">{line}</span>
      {:else if r.problem === 'failed' && s}
        <span class="stale" title={d.failed}>!</span>
      {/if}
    </div>
    {#if message}
      <p class="message">{message}</p>
    {:else if s}
      <div class="big">
        {s.running}<small>/{s.total} {d.running}</small>
      </div>
      <div class="dots">
        <Dots cells={dots} cols={COLS} size={9} gap={3} colors={COLORS} />
        {#if size === 'S' && s.total > COLS}<span class="more">+{s.total - COLS}</span>{/if}
      </div>
      {#if size === 'M' && names.length > 0}
        <ul>
          {#each names.slice(0, NAMED) as n (n.kind + n.name)}
            <li>
              <span class="dot"></span><span class="name">{n.name}</span>
              <span class="why">{d.states[n.kind]}</span>
            </li>
          {/each}
          {#if names.length + s.more > NAMED}
            <li class="more">{d.more(names.length + s.more - NAMED)}</li>
          {/if}
        </ul>
      {/if}
    {/if}
  </div>
</WidgetCard>

<style>
  .docker {
    display: flex;
    flex-direction: column;
    gap: 4px;
    height: 100%;
    min-height: 0;
  }

  .head {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 8px;
    min-width: 0;
  }

  .title {
    display: flex;
    flex: none;
    align-items: center;
    gap: 6px;
  }

  .label {
    flex: none;
    color: var(--fg-on-dark-muted);
    font-size: var(--text-xxs);
    font-weight: 600;
    letter-spacing: 0.05em;
    text-transform: uppercase;
  }

  .state {
    overflow: hidden;
    color: var(--fg-on-dark-muted);
    font-size: var(--text-xs);
    font-weight: 600;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .state.ok {
    color: var(--ok-on-dark);
  }

  .state.warn {
    color: var(--warn-on-dark);
  }

  .state.danger {
    color: var(--danger-on-dark);
  }

  .stale {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 14px;
    height: 14px;
    border-radius: 50%;
    background: var(--warn-on-dark);
    color: var(--widget-base);
    font-size: var(--text-xxs);
    font-weight: 700;
  }

  .big {
    color: var(--fg-on-dark);
    font-size: 30px;
    font-variant-numeric: tabular-nums;
    font-weight: 650;
    line-height: 1.1;
  }

  .big small {
    margin-left: 3px;
    color: var(--fg-on-dark-muted);
    font-size: var(--text-xs);
    font-weight: 500;
  }

  .dots {
    display: flex;
    align-items: flex-end;
    gap: 6px;
    margin-top: 2px;
  }

  .more {
    color: var(--fg-on-dark-muted);
    font-size: var(--text-xs);
  }

  ul {
    display: flex;
    flex-direction: column;
    gap: 3px;
    margin: 6px 0 0;
    padding: 0;
    list-style: none;
    font-size: var(--text-xs);
  }

  li {
    display: flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
  }

  li .dot {
    flex: none;
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: var(--danger-on-dark);
  }

  .name {
    overflow: hidden;
    color: var(--fg-on-dark);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .why {
    margin-left: auto;
    color: var(--fg-on-dark-muted);
    white-space: nowrap;
  }

  li.more {
    color: var(--fg-on-dark-muted);
  }

  .message {
    margin: auto 0;
    color: var(--fg-on-dark-muted);
    font-size: var(--text-xs);
  }
</style>
