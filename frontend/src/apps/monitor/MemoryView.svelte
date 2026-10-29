<script lang="ts">
  import { formatBytes, formatPct } from '../../lib/format'
  import { t } from '../../lib/i18n'
  import Meter from '../../lib/ui/charts/Meter.svelte'
  import { level } from '../../lib/ui/charts/scale'
  import StackBar from '../../lib/ui/charts/StackBar.svelte'
  import TimeChart from '../../lib/ui/charts/TimeChart.svelte'
  import Card from './Card.svelte'
  import Hero from './Hero.svelte'
  import Stat from './Stat.svelte'
  import { chartColors, memPct, type ViewProps } from './types'

  let { data, range, rangeLabel }: ViewProps = $props()

  const m = $derived(data.latest)
  const pct = $derived(memPct(m.memUsed, m.memTotal))
  // "Used" already excludes reclaimable cache (like free), so the rest of
  // the bar is what nothing holds at all.
  const cache = $derived(Math.min(m.memCache, Math.max(0, m.memTotal - m.memUsed)))
  const free = $derived(Math.max(0, m.memTotal - m.memUsed - cache))
  const swapPct = $derived(memPct(m.swapUsed, m.swapTotal))
</script>

<div class="stack">
  <Card title={t.monitor.memory} icon="memory">
    <div class="head">
      <Hero
        value={formatBytes(m.memUsed)}
        caption="{formatPct(pct)} · {t.monitor.ofTotal(formatBytes(m.memTotal))}"
        tone={level(pct, 90, 95)}
      />
      <div class="split">
        <Stat label={t.monitor.used} value={formatBytes(m.memUsed)} color={chartColors[0]} />
        <Stat label={t.monitor.cache} value={formatBytes(cache)} color={chartColors[2]} />
        <Stat label={t.monitor.free} value={formatBytes(free)} color="var(--fill-track)" />
      </div>
    </div>
    <StackBar
      total={m.memTotal}
      label="{t.monitor.used} {formatBytes(m.memUsed)}, {t.monitor.cache} {formatBytes(cache)}, {t
        .monitor.free} {formatBytes(free)}"
      segments={[
        { label: t.monitor.used, value: m.memUsed, color: chartColors[0] },
        { label: t.monitor.cache, value: cache, color: chartColors[2] },
      ]}
    />
    <TimeChart
      samples={data.history}
      {range}
      end={m.at}
      {rangeLabel}
      label={t.monitor.memory}
      series={[
        { label: t.monitor.used, color: chartColors[0], value: (s) => s.memUsed },
        { label: t.monitor.cache, color: chartColors[2], value: (s) => s.memCache },
      ]}
      max={m.memTotal}
      format={formatBytes}
      height={150}
    />
  </Card>

  <Card title={t.monitor.swap} icon="disk">
    {#if m.swapTotal > 0}
      <div class="swap">
        <Stat
          label={t.monitor.used}
          value="{formatBytes(m.swapUsed)} / {formatBytes(m.swapTotal)}"
        />
        <span class="pct">{formatPct(swapPct)}</span>
      </div>
      <Meter pct={swapPct} label={t.monitor.swap} warn={50} danger={80} />
    {:else}
      <p class="muted">{t.monitor.noSwap}</p>
    {/if}
  </Card>
</div>

<style>
  .stack {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .head {
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
    gap: 20px;
    flex-wrap: wrap;
  }

  .split {
    display: flex;
    gap: 22px;
    flex-wrap: wrap;
  }

  .swap {
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
  }

  .pct {
    font-size: var(--text-md);
    font-weight: 600;
    font-variant-numeric: tabular-nums;
  }

  .muted {
    margin: 0;
    font-size: var(--text-sm);
    color: var(--fg-muted);
  }
</style>
