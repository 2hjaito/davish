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

  // A ring in the bar; on hover, the value large with the last
  // 15 minutes under it, like the Monitor's cards.
  const R = 7
  const C = 2 * Math.PI * R
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
  role="img"
  aria-label="{view.label}: {view.text}"
  onmouseenter={show}
  onmouseleave={hide}
>
  {#if view.ring !== null}
    <svg class="ring" viewBox="0 0 18 18" aria-hidden="true">
      <circle class="track" cx="9" cy="9" r={R} />
      <circle
        class="fill"
        cx="9"
        cy="9"
        r={R}
        stroke-dasharray={C}
        stroke-dashoffset={C * (1 - view.ring)}
      />
    </svg>
  {:else}
    <Icon name={view.icon} size={13} strokeWidth={2.2} />
  {/if}
  {#if view.short}<span class="label">{view.short}</span>{/if}
  <span class="value">{view.text}</span>
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
  .stat {
    height: var(--menubar-h);
    display: flex;
    align-items: center;
    gap: 5px;
    white-space: nowrap;
    border-radius: 6px;
  }

  /* The bar's readings are where a frameless window is grabbed to move it;
     the menu bar resets everything else to no-drag. */
  :global(.menubar) .stat,
  :global(.menubar) .stat :global(*) {
    --wails-draggable: drag;
  }

  .label {
    font-size: var(--text-xs);
    color: var(--fg-label);
  }

  .ring {
    width: 16px;
    height: 16px;
    rotate: -90deg;
    flex-shrink: 0;
  }

  .ring circle {
    fill: none;
    stroke-width: 2.6;
  }

  .track {
    stroke: var(--fill-track);
  }

  .fill {
    stroke: var(--accent);
    stroke-linecap: round;
    transition: stroke-dashoffset var(--dur-slow) var(--ease-out);
  }

  .warn {
    color: var(--warn);
  }

  .warn .fill {
    stroke: var(--warn);
  }

  .danger {
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
