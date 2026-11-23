<script lang="ts">
  import { untrack } from 'svelte'

  import { hasRuntime } from '../api/runtime'
  import { t } from '../i18n'
  import { dockerHealth } from '../stores/dockerHealth.svelte'
  import { servers } from '../stores/servers.svelte'
  import { hasProblem, problemCount } from './dockerHealth'
  import type { WidgetProps } from './registry'

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
  const level = $derived(bad ? 'danger' : s && s.starting > 0 ? 'warn' : 'ok')

  /** One line on the state: what is wrong, else what is starting, else that all is well. */
  const line = $derived.by(() => {
    if (!s) return ''
    if (s.total === 0) return d.none
    const parts: string[] = []
    if (s.unhealthy.length + s.more > 0 && s.restarting.length === 0)
      parts.push(d.unhealthy(problemCount(s)))
    else {
      if (s.unhealthy.length > 0) parts.push(d.unhealthy(s.unhealthy.length))
      if (s.restarting.length > 0) parts.push(d.restarting(s.restarting.length + s.more))
    }
    if (parts.length === 0 && s.starting > 0) parts.push(d.starting(s.starting))
    return parts.length > 0 ? parts.join(' · ') : d.allHealthy
  })

  /** The containers to name in the medium widget: the failing ones first. */
  const NAMED = 4
  const names = $derived(
    s
      ? [
          ...s.unhealthy.map((name) => ({ name, kind: 'unhealthy' })),
          ...s.restarting.map((name) => ({ name, kind: 'restarting' })),
        ]
      : [],
  )
</script>

<div class="docker">
  <div class="head">
    <span class="label">{t.apps.docker}</span>
    {#if r.problem === 'failed' && s}<span class="stale" title={d.failed}>!</span>{/if}
  </div>
  {#if message}
    <p class="message">{message}</p>
  {:else if s}
    <div class="big">
      {s.running}<small>/{s.total} {d.running}</small>
    </div>
    <div class="line {level}"><span class="dot"></span>{line}</div>
    {#if size === 'M' && names.length > 0}
      <ul>
        {#each names.slice(0, NAMED) as n (n.kind + n.name)}
          <li class={n.kind}><span class="dot"></span><span class="name">{n.name}</span></li>
        {/each}
        {#if names.length + s.more > NAMED}
          <li class="more">{d.more(names.length + s.more - NAMED)}</li>
        {/if}
      </ul>
    {/if}
  {/if}
</div>

<style>
  .docker {
    display: flex;
    flex-direction: column;
    gap: 2px;
    height: 100%;
    min-height: 0;
    overflow: hidden;
  }

  .head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    color: var(--fg-muted);
    font-size: var(--text-xs);
  }

  .stale {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 14px;
    height: 14px;
    border-radius: 50%;
    background: var(--warn);
    color: var(--fg-on-accent);
    font-size: var(--text-xxs);
    font-weight: 700;
  }

  .big {
    color: var(--fg);
    font-size: 24px;
    font-variant-numeric: tabular-nums;
    font-weight: 600;
    line-height: 1.15;
  }

  .big small {
    margin-left: 2px;
    color: var(--fg-muted);
    font-size: var(--text-xs);
    font-weight: 400;
  }

  .line {
    display: flex;
    align-items: center;
    gap: 6px;
    overflow: hidden;
    font-size: var(--text-xs);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .dot {
    flex: none;
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: var(--fg-muted);
  }

  .ok .dot {
    background: var(--ok);
  }

  .warn .dot {
    background: var(--warn);
  }

  .danger {
    color: var(--danger);
  }

  .danger .dot {
    background: var(--danger);
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

  li.unhealthy .dot,
  li.restarting .dot {
    background: var(--danger);
  }

  .name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .more {
    color: var(--fg-muted);
  }

  .message {
    margin: auto 0;
    color: var(--fg-muted);
    font-size: var(--text-xs);
  }
</style>
