<script lang="ts" generics="T extends { at: number }">
  import { t } from '../../i18n/vi'
  import { areaPath, type ChartSeries, linePath, nearestIndex, niceMax } from './scale'

  interface Props {
    samples: T[]
    series: ChartSeries<T>[]
    /** Visible span in ms, ending at end. */
    range: number
    end: number
    /** "5 phút": the left edge reads "5 phút trước". */
    rangeLabel: string
    format: (v: number) => string
    /** Fixed top of the axis (100 for percentages); otherwise fitted to the data. */
    max?: number
    /** Round the fitted top to powers of 1024 (byte rates). */
    binary?: boolean
    /** Lowest fitted top, so an idle line does not fill the chart with noise. */
    floor?: number
    height?: number
    /** No axes, labels or end dots: a sparkline for overview cards. */
    compact?: boolean
    /** Names the chart for screen readers. */
    label: string
  }

  let {
    samples,
    series,
    range,
    end,
    rangeLabel,
    format,
    max,
    binary = false,
    floor = 1,
    height = 120,
    compact = false,
    label,
  }: Props = $props()

  // Three missed ticks (collector at 2s) count as a hole in the data.
  const MAX_GAP = 6500

  let width = $state(0)
  let hover = $state<number | null>(null)

  const padTop = $derived(compact ? 3 : 16)
  const padBottom = $derived(compact ? 1 : 18)
  const plotH = $derived(Math.max(1, height - padTop - padBottom))
  const start = $derived(end - range)

  // One sample before the left edge keeps the line running into the chart.
  const visible = $derived.by(() => {
    const first = samples.findIndex((s) => s.at >= start)
    if (first < 0) return []
    return samples.slice(Math.max(0, first - 1))
  })
  const times = $derived(visible.map((s) => s.at))

  const top = $derived.by(() => {
    if (max !== undefined) return max
    let m = 0
    for (const s of visible) for (const se of series) m = Math.max(m, se.value(s))
    return niceMax(m, { binary, floor })
  })

  const x = (at: number) => ((at - start) / range) * width
  const y = (v: number) => padTop + plotH - (Math.min(Math.max(v, 0), top) / top) * plotH

  const paths = $derived(
    series.map((se) => {
      const pts = visible.map((s) => ({ x: x(s.at), y: y(se.value(s)) }))
      return {
        color: se.color,
        line: linePath(pts, times, MAX_GAP),
        area: areaPath(pts, times, MAX_GAP, padTop + plotH),
        last: pts.at(-1),
      }
    }),
  )

  const hovered = $derived(hover === null ? undefined : visible[hover])

  function onPointerMove(e: PointerEvent) {
    const box = (e.currentTarget as HTMLElement).getBoundingClientRect()
    const i = nearestIndex(times, start + ((e.clientX - box.left) / box.width) * range)
    hover = i >= 0 && (times[i] ?? 0) >= start ? i : null
  }

  function onKeyDown(e: KeyboardEvent) {
    if (visible.length === 0) return
    const last = visible.length - 1
    if (e.key === 'ArrowLeft') hover = Math.max(0, (hover ?? last + 1) - 1)
    else if (e.key === 'ArrowRight') hover = Math.min(last, (hover ?? last - 1) + 1)
    else if (e.key === 'Escape') hover = null
    else return
    e.preventDefault()
  }

  function clock(at: number) {
    const d = new Date(at)
    const pad = (n: number) => String(n).padStart(2, '0')
    return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
  }
</script>

<!-- The chart is one focus stop; arrow keys walk the samples like the pointer
     does. No ARIA role fits that and also counts as interactive to Svelte. -->
<!-- svelte-ignore a11y_no_noninteractive_tabindex, a11y_no_noninteractive_element_interactions -->
<div
  class="chart"
  class:compact
  style:height="{height}px"
  bind:clientWidth={width}
  role="application"
  aria-label="{label}. {t.monitor.chartHint}"
  tabindex={compact ? -1 : 0}
  onpointermove={compact ? undefined : onPointerMove}
  onpointerleave={() => (hover = null)}
  onkeydown={compact ? undefined : onKeyDown}
  onblur={() => (hover = null)}
>
  {#if width > 0}
    <svg {width} {height} aria-hidden="true">
      {#if !compact}
        {#each [top, top / 2] as v (v)}
          <line class="grid" x1="0" x2={width} y1={y(v)} y2={y(v)} />
          <text class="tick" x="0" y={y(v) - 4}>{format(v)}</text>
        {/each}
      {/if}
      <line class="grid base" x1="0" x2={width} y1={padTop + plotH} y2={padTop + plotH} />

      {#each paths as p, i (i)}
        <path d={p.area} fill={p.color} class="area" />
        <path d={p.line} stroke={p.color} class="line" class:thin={compact} />
      {/each}

      {#if hovered}
        <line
          class="crosshair"
          x1={x(hovered.at)}
          x2={x(hovered.at)}
          y1={padTop}
          y2={padTop + plotH}
        />
        {#each series as se, i (i)}
          <circle class="dot" cx={x(hovered.at)} cy={y(se.value(hovered))} r="4" fill={se.color} />
        {/each}
      {:else if !compact}
        {#each paths as p, i (i)}
          {#if p.last}
            <circle class="dot" cx={p.last.x} cy={p.last.y} r="4" fill={p.color} />
          {/if}
        {/each}
      {/if}

      {#if !compact}
        <text class="tick" x="0" y={height - 4}>{t.monitor.ago(rangeLabel)}</text>
        <text class="tick" x={width} y={height - 4} text-anchor="end">{t.monitor.now}</text>
      {/if}
    </svg>
  {/if}

  {#if hovered}
    <div
      class="tip"
      class:flip={x(hovered.at) > width / 2}
      style:left="{x(hovered.at)}px"
      style:top="{padTop}px"
    >
      <span class="time">{clock(hovered.at)}</span>
      {#each series as se, i (i)}
        <span class="row">
          <span class="key" style:background={se.color}></span>
          <strong>{format(se.value(hovered))}</strong>
          {#if series.length > 1}<span class="name">{se.label}</span>{/if}
        </span>
      {/each}
    </div>
  {/if}
</div>

<style>
  .chart {
    position: relative;
    width: 100%;
    outline: none;
    border-radius: 6px;
  }

  .chart:not(.compact) {
    cursor: crosshair;
  }

  .chart:focus-visible {
    box-shadow: 0 0 0 2px var(--accent-soft);
  }

  svg {
    display: block;
    overflow: visible;
  }

  .grid {
    stroke: var(--chart-grid);
    stroke-width: 1;
    shape-rendering: crispEdges;
  }

  .tick {
    font-size: var(--text-xxs);
    fill: var(--fg-subtle);
    font-variant-numeric: tabular-nums;
  }

  .area {
    fill-opacity: 0.1;
    stroke: none;
  }

  .line {
    fill: none;
    stroke-width: 2;
    stroke-linejoin: round;
    stroke-linecap: round;
  }

  .line.thin {
    stroke-width: 1.5;
  }

  .crosshair {
    stroke: var(--chart-crosshair);
    stroke-width: 1;
    shape-rendering: crispEdges;
  }

  .dot {
    stroke: var(--chart-surface);
    stroke-width: 2;
  }

  .tip {
    position: absolute;
    z-index: 1;
    transform: translateX(10px);
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 96px;
    padding: 6px 9px;
    border-radius: 8px;
    background: var(--glass-bg-menu);
    -webkit-backdrop-filter: var(--glass-blur);
    backdrop-filter: var(--glass-blur);
    box-shadow: var(--glass-shadow);
    font-size: var(--text-xs);
    pointer-events: none;
    white-space: nowrap;
  }

  .tip.flip {
    transform: translateX(calc(-100% - 10px));
  }

  .time {
    color: var(--fg-subtle);
    font-variant-numeric: tabular-nums;
  }

  .row {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .row strong {
    font-variant-numeric: tabular-nums;
    color: var(--fg);
  }

  .key {
    width: 10px;
    height: 2px;
    border-radius: 1px;
    flex-shrink: 0;
  }

  .name {
    color: var(--fg-muted);
  }
</style>
