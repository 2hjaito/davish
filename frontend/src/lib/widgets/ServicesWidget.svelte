<script lang="ts">
  import { untrack } from 'svelte'

  import { hasRuntime } from '../api/runtime'
  import { t } from '../i18n'
  import { servers } from '../stores/servers.svelte'
  import { servicesHealth } from '../stores/servicesHealth.svelte'
  import Bar from './Bar.svelte'
  import type { WidgetProps } from './registry'
  import { hasProblem, type Group, problemCount } from './servicesHealth'
  import WidgetCard, { type Tone } from './WidgetCard.svelte'

  let { serverID, size }: WidgetProps = $props()

  const d = t.widgets.services

  const connected = $derived(
    !hasRuntime || servers.list.find((s) => s.id === serverID)?.state === 'connected',
  )

  // One `systemctl list-units` a minute for the server, shared with every
  // Services widget watching it (stores/polledReading.svelte.ts); refresh
  // reads state of its own, which must not become a dependency here.
  $effect(() => servicesHealth.watch(serverID))
  $effect(() => {
    if (connected) untrack(() => void servicesHealth.refresh(serverID))
  })

  const r = $derived(servicesHealth.of(serverID))
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
    !s || message ? 'neutral' : bad ? 'danger' : s.counts.starting > 0 ? 'warn' : 'ok',
  )

  /** A short word for the state: how many problems, else what is starting, else that all is well. */
  const line = $derived.by(() => {
    if (!s) return ''
    if (s.total === 0) return d.none
    if (bad) return d.problems(problemCount(s))
    if (s.counts.starting > 0) return d.starting(s.counts.starting)
    return d.allRunning
  })

  const COLORS: Record<Group, string> = {
    running: 'var(--ok-on-dark)',
    exited: 'var(--widget-blue)',
    starting: 'var(--warn-on-dark)',
    bad: 'var(--danger-on-dark)',
    inactive: 'var(--widget-stopped)',
  }

  /** The bar in the order of how much it matters: well first, then waiting, then wrong. */
  const ORDER: Group[] = ['running', 'exited', 'starting', 'bad', 'inactive']
  const parts = $derived(s ? ORDER.map((kind) => ({ kind, count: s.counts[kind] })) : [])

  const NAMED = 3
  const named = $derived(s ? s.problems.slice(0, NAMED) : [])
  const rest = $derived(s ? s.problems.length + s.more - named.length : 0)
</script>

<WidgetCard {tone}>
  <div class="services">
    <div class="head">
      <span class="label">{t.apps.services}</span>
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
        {s.counts.running}<small>{d.running}</small>
      </div>
      <Bar {parts} colors={COLORS} height={size === 'M' ? 9 : 7} />
      {#if size === 'M'}
        <p class="rest">
          {d.others(s.counts.exited, s.counts.inactive)}
        </p>
        {#if named.length > 0}
          <ul>
            {#each named as p (p.name)}
              <li>
                <span class="dot"></span><span class="name">{p.name}</span>
                <span class="why">{d.why[p.why]}</span>
              </li>
            {/each}
            {#if rest > 0}<li class="more">{d.more(rest)}</li>{/if}
          </ul>
        {/if}
      {/if}
    {/if}
  </div>
</WidgetCard>

<style>
  .services {
    display: flex;
    flex-direction: column;
    gap: 5px;
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
    margin-left: 4px;
    color: var(--fg-on-dark-muted);
    font-size: var(--text-xs);
    font-weight: 500;
  }

  .rest {
    margin: 0;
    color: var(--fg-on-dark-muted);
    font-size: var(--text-xs);
  }

  ul {
    display: flex;
    flex-direction: column;
    gap: 3px;
    margin: 4px 0 0;
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
