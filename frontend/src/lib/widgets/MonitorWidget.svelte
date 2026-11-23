<script lang="ts">
  import { t } from '../i18n'
  import { statView, type StatId } from '../shell/menuStats'
  import { metrics } from '../stores/metrics.svelte'
  import { niceMax } from '../ui/charts/scale'
  import type { WidgetProps } from './registry'
  import Spark from './Spark.svelte'

  let { serverID, size }: WidgetProps = $props()

  // Shares the collector with the menu bar: one watch more, no command more.
  $effect(() => metrics.watch(serverID))

  const latest = $derived(metrics.latest(serverID))
  const history = $derived(metrics.history(serverID))
  const ids: StatId[] = $derived(size === 'M' ? ['cpu', 'mem', 'net'] : ['cpu'])

  const rows = $derived(
    latest
      ? ids.flatMap((id) => {
          const v = statView(id, latest)
          if (!v?.chart) return []
          const series = v.chart.series[0]
          if (!series) return []
          const values = history.map((s) => series.value(s))
          const max = v.chart.max ?? niceMax(Math.max(...values, 0), { binary: v.chart.binary })
          return [{ v, values, max, color: series.color, times: history.map((s) => s.at) }]
        })
      : [],
  )
</script>

<div class="monitor" class:big={size === 'M'}>
  {#each rows as r (r.v.id)}
    <div class="row {r.v.level}">
      <div class="text">
        <span class="label">{r.v.label}</span>
        <span class="hero">{r.v.hero}</span>
      </div>
      <div class="spark">
        <Spark values={r.values} times={r.times} max={r.max} color={r.color} />
      </div>
    </div>
  {:else}
    <p class="wait">{t.widgets.waiting}</p>
  {/each}
</div>

<style>
  .monitor {
    display: flex;
    flex-direction: column;
    justify-content: space-around;
    height: 100%;
    min-height: 0;
  }

  .row {
    display: flex;
    align-items: center;
    gap: 10px;
    min-height: 0;
    flex: 1;
  }

  .text {
    display: flex;
    flex-direction: column;
    min-width: 0;
    flex: 0 0 auto;
    width: 72px;
  }

  .label {
    color: var(--fg-muted);
    font-size: var(--text-xs);
  }

  .hero {
    overflow: hidden;
    color: var(--fg);
    font-size: var(--text-lg);
    font-variant-numeric: tabular-nums;
    font-weight: 600;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .warn .hero {
    color: var(--warn);
  }

  .danger .hero {
    color: var(--danger);
  }

  .spark {
    flex: 1;
    min-width: 0;
    height: 26px;
  }

  .wait {
    margin: auto;
    color: var(--fg-muted);
    font-size: var(--text-xs);
  }
</style>
