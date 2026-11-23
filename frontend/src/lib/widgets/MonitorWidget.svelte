<script lang="ts">
  import { t } from '../i18n'
  import { loadFormat, statView, type StatId, type StatView } from '../shell/menuStats'
  import { metrics } from '../stores/metrics.svelte'
  import { niceMax } from '../ui/charts/scale'
  import Dots from './Dots.svelte'
  import { cellsFor, dotCells, splitUnit } from './gauge'
  import type { WidgetProps } from './registry'
  import Ring from './Ring.svelte'
  import { LOOK } from './look'
  import Spark from './Spark.svelte'
  import WidgetBadge from './WidgetBadge.svelte'
  import WidgetCard, { type Tone } from './WidgetCard.svelte'

  let { serverID, size }: WidgetProps = $props()

  // Shares the collector with the menu bar: one watch more, no command more.
  $effect(() => metrics.watch(serverID))

  const latest = $derived(metrics.latest(serverID))
  const history = $derived(metrics.history(serverID))
  const times = $derived(history.map((s) => s.at))

  const view = (id: StatId): StatView | null => (latest ? statView(id, latest) : null)
  const cpu = $derived(view('cpu'))
  /** The cores, their number large and the word small; the CPU share is in the ring. */
  const cores = $derived(splitUnit(t.monitor.cores(latest?.cores.length ?? 0)))
  const load = $derived(latest ? `${t.menubar.stats.names.load} ${loadFormat(latest.load1)}` : '')
  const mem = $derived(view('mem'))
  const net = $derived(view('net'))

  const COLOR = {
    '': 'var(--widget-blue)',
    warn: 'var(--warn-on-dark)',
    danger: 'var(--danger-on-dark)',
  }
  const MEM_COLOR = {
    '': 'var(--widget-violet)',
    warn: 'var(--warn-on-dark)',
    danger: 'var(--danger-on-dark)',
  }

  /** The values of series i of a reading's chart over the history. */
  const values = (v: StatView | null, i: number) => {
    const series = v?.chart?.series[i]
    return series ? history.map((s) => series.value(s)) : []
  }

  const cpuValues = $derived(values(cpu, 0))
  const rxValues = $derived(values(net, 0))
  const txValues = $derived(values(net, 1))
  const netMax = $derived(niceMax(Math.max(...rxValues, ...txValues, 0), { binary: true }))

  const tone = $derived<Tone>(
    cpu?.level === 'danger' || mem?.level === 'danger'
      ? 'danger'
      : cpu?.level === 'warn' || mem?.level === 'warn'
        ? 'warn'
        : 'neutral',
  )

  // Memory as a grid of dots: the used share lit, the rest dim.
  const COLS = 15
  const ROWS = 3
  const memCells = $derived(
    mem?.ring == null
      ? []
      : dotCells(COLS * ROWS, [{ kind: 'used', count: cellsFor(mem.ring, COLS * ROWS) }]),
  )
</script>

<WidgetCard {tone} accent={LOOK.monitor.accent}>
  {#if !cpu}
    <p class="wait">{t.widgets.waiting}</p>
  {:else if size === 'S'}
    <div class="chart">
      <Spark
        series={[{ values: cpuValues, color: COLOR[cpu.level], area: true }]}
        {times}
        max={100}
      />
    </div>
    <div class="s">
      <div class="text">
        <span class="title"
          ><WidgetBadge kind="monitor" /><span class="label">{cpu.label}</span></span
        >
        <span class="hero big">{cores.value}<small>{cores.unit}</small></span>
        <span class="sub">{load}</span>
      </div>
      <Ring value={cpu.ring ?? 0} size={60} stroke={7} color={COLOR[cpu.level]}>
        <span class="inner">{cpu.hero}</span>
      </Ring>
    </div>
  {:else}
    <div class="m">
      <div class="top">
        <div class="text">
          <span class="title"
            ><WidgetBadge kind="monitor" /><span class="label">{cpu.label}</span></span
          >
          <span class="hero">{cores.value}<small>{cores.unit}</small></span>
          <span class="sub">{load}</span>
        </div>
        <Ring value={cpu.ring ?? 0} size={54} stroke={6} color={COLOR[cpu.level]}>
          <span class="inner">{cpu.hero}</span>
        </Ring>
      </div>
      {#if mem}
        <div class="block" title={mem.sub}>
          <div class="line">
            <span class="label">{mem.label}</span>
            <span class="hero small">{mem.hero}</span>
          </div>
          <Dots
            cells={memCells}
            cols={COLS}
            size={7}
            gap={2.4}
            colors={{ used: MEM_COLOR[mem.level] }}
          />
        </div>
      {/if}
      {#if net}
        <div class="block" title={net.sub}>
          <div class="line">
            <span class="label">{net.label}</span>
            <span class="hero small">↓ {net.hero}</span>
          </div>
          <div class="chart inline">
            <Spark
              series={[
                { values: rxValues, color: 'var(--widget-blue)', area: true },
                { values: txValues, color: 'var(--widget-pink)' },
              ]}
              {times}
              max={netMax}
            />
          </div>
        </div>
      {/if}
    </div>
  {/if}
</WidgetCard>

<style>
  .s {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 100%;
  }

  .text {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }

  .title {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-bottom: 2px;
  }

  .label {
    color: var(--fg-on-dark-muted);
    font-size: var(--text-xxs);
    font-weight: 600;
    letter-spacing: 0.05em;
    text-transform: uppercase;
  }

  .hero {
    color: var(--fg-on-dark);
    font-size: 24px;
    font-variant-numeric: tabular-nums;
    font-weight: 650;
    line-height: 1.1;
    white-space: nowrap;
  }

  .hero small {
    margin-left: 3px;
    color: var(--fg-on-dark-muted);
    font-size: var(--text-sm);
    font-weight: 500;
  }

  .inner {
    color: var(--fg-on-dark);
    font-size: var(--text-xs);
    font-weight: 650;
  }

  .hero.big {
    font-size: 30px;
  }

  .hero.small {
    font-size: 17px;
  }

  .sub {
    overflow: hidden;
    color: var(--fg-on-dark-muted);
    font-size: var(--text-xs);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  /* The history, faint, along the bottom of the small widget, below the figure. */
  .chart {
    position: absolute;
    right: 0;
    bottom: 0;
    left: 0;
    height: 22px;
    opacity: 0.55;
    pointer-events: none;
  }

  .chart.inline {
    position: static;
    height: 24px;
    margin-top: 1px;
    opacity: 1;
  }

  .m {
    display: flex;
    flex-direction: column;
    gap: 7px;
    height: 100%;
    min-height: 0;
  }

  .top {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .block {
    display: flex;
    flex-direction: column;
    gap: 3px;
    min-width: 0;
  }

  .line {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 8px;
  }

  .wait {
    margin: auto 0;
    color: var(--fg-on-dark-muted);
    font-size: var(--text-xs);
  }
</style>
