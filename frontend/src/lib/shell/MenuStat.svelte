<script lang="ts">
  import { t } from '../i18n'
  import type { Sample } from '../stores/metrics.svelte'
  import TimeChart from '../ui/charts/TimeChart.svelte'
  import Icon from '../ui/Icon.svelte'
  import { HISTORY_MS } from '../stores/metrics.svelte'
  import type { StatView } from './menuStats'

  interface Props {
    view: StatView
    history: Sample[]
    /** When the latest sample arrived; the chart ends there. */
    end: number
  }

  let { view, history, end }: Props = $props()

  // A ring in the bar with the number in it; on hover, the value large
  // with the last 15 minutes under it, like the Monitor's cards.
  const R = 7.75
  const C = 2 * Math.PI * R
  // A gauge open at the bottom: three quarters of a turn.
  const ARC = C * 0.75
  const POP_W = 300
  const SHOW_MS = 120
  const HIDE_MS = 150

  let anchor = $state<HTMLElement>()
  let open = $state(false)
  let pos = $state({ x: 0, y: 0 })
  let timer = 0

  function show() {
    clearTimeout(timer)
    timer = window.setTimeout(() => {
      if (!anchor) return
      const r = anchor.getBoundingClientRect()
      const x = r.left + r.width / 2 - POP_W / 2
      pos = { x: Math.min(Math.max(8, x), window.innerWidth - POP_W - 8), y: r.bottom + 6 }
      open = true
    }, SHOW_MS)
  }

  function hide() {
    clearTimeout(timer)
    timer = window.setTimeout(() => (open = false), HIDE_MS)
  }

  $effect(() => () => clearTimeout(timer))
</script>

<span
  bind:this={anchor}
  class="stat {view.level}"
  class:wide={!view.inner}
  role="img"
  aria-label="{view.label}: {view.text}"
  onmouseenter={show}
  onmouseleave={hide}
>
  <span class="ring">
    <svg viewBox="0 0 18 18" aria-hidden="true">
      <circle class="track" cx="9" cy="9" r={R} stroke-dasharray="{ARC} {C}" />
      {#if view.ring !== null}
        <circle class="fill" cx="9" cy="9" r={R} stroke-dasharray="{ARC * view.ring} {C}" />
      {/if}
    </svg>
    {#if view.inner}
      <span class="inner" class:long={view.inner.length > 2}>{view.inner}</span>
    {:else}
      <Icon name={view.icon} size={9} strokeWidth={2.4} />
    {/if}
  </span>
  <span class="label">{view.inner ? view.short : view.text}</span>
</span>

{#if open}
  <div
    class="pop glass glass-strong {view.level}"
    role="tooltip"
    style:left="{pos.x}px"
    style:top="{pos.y}px"
    style:width="{POP_W}px"
    onmouseenter={() => clearTimeout(timer)}
    onmouseleave={hide}
  >
    <div class="head">
      <span class="hero">{view.hero}</span>
      <span class="sub">{view.label} · {view.sub}</span>
    </div>
    {#if view.chart}
      <TimeChart
        samples={history}
        series={view.chart.series}
        range={HISTORY_MS}
        {end}
        rangeLabel={t.menubar.stats.chartRange}
        format={view.chart.format}
        max={view.chart.max}
        binary={view.chart.binary}
        floor={view.chart.floor}
        height={110}
        label={view.label}
      />
    {/if}
  </div>
{/if}

<style>
  /* The ring with its name under it, as on a phone's battery widget. */
  /* A fixed width, so a value that grows or shrinks does not shift the
     readings beside it; wider where the value sits under the ring. */
  .stat {
    width: 30px;
    height: var(--menubar-h);
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 2px;
    white-space: nowrap;
    border-radius: 6px;
  }

  .stat.wide {
    width: 44px;
  }

  /* The bar's readings are where a frameless window is grabbed to move it;
     the menu bar resets everything else to no-drag. */
  :global(.menubar) .stat,
  :global(.menubar) .stat :global(*) {
    --wails-draggable: drag;
  }

  .label {
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    font-size: 8.5px;
    line-height: 1;
    color: var(--fg-label);
  }

  .ring {
    position: relative;
    width: 18px;
    height: 18px;
    display: grid;
    place-items: center;
    flex-shrink: 0;
  }

  .ring svg {
    position: absolute;
    inset: 0;
    /* Starts the arc at bottom left, so the gap sits under the number. */
    rotate: 135deg;
  }

  .ring circle {
    fill: none;
    stroke-width: 1.9;
    stroke-linecap: round;
  }

  .inner {
    position: relative;
    font-size: 7.5px;
    font-weight: 600;
    letter-spacing: -0.02em;
    line-height: 1;
  }

  /* 100 and loads like 2.4 need a little less to stay inside the ring. */
  .inner.long {
    font-size: 6px;
    letter-spacing: -0.04em;
  }

  .track {
    stroke: var(--fill-track);
  }

  .fill {
    stroke: var(--ok);
    transition: stroke-dasharray var(--dur-slow) var(--ease-out);
  }

  .warn .inner {
    color: var(--warn);
  }

  .warn .fill {
    stroke: var(--warn);
  }

  .danger .inner {
    color: var(--danger);
  }

  .danger .fill {
    stroke: var(--danger);
  }

  .pop {
    position: fixed;
    z-index: 1100;
    padding: 12px 14px 10px;
    border-radius: var(--menu-radius);
    background: var(--glass-bg-menu);
    color: var(--fg);
    white-space: normal;
  }

  .head {
    display: flex;
    align-items: baseline;
    flex-wrap: wrap;
    column-gap: 8px;
    margin-bottom: 4px;
  }

  .hero {
    font-size: 22px;
    font-weight: 600;
  }

  .pop.warn .hero {
    color: var(--warn);
  }

  .pop.danger .hero {
    color: var(--danger);
  }

  .sub {
    font-size: var(--text-sm);
    color: var(--fg-muted);
  }
</style>
