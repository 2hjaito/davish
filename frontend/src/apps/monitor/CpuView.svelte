<script lang="ts">
  import { formatPct } from '../../lib/format'
  import { t } from '../../lib/i18n'
  import Meter from '../../lib/ui/charts/Meter.svelte'
  import { level } from '../../lib/ui/charts/scale'
  import TimeChart from '../../lib/ui/charts/TimeChart.svelte'
  import Card from './Card.svelte'
  import Hero from './Hero.svelte'
  import Stat from './Stat.svelte'
  import { chartColors, type ViewProps } from './types'

  let { data, range, rangeLabel }: ViewProps = $props()

  const m = $derived(data.latest)
  const loadFormat = (v: number) => v.toFixed(v < 10 ? 2 : 1)
</script>

<div class="stack">
  <Card title={t.monitor.cpu} icon="cpu">
    <div class="head">
      <Hero
        value={formatPct(m.cpu)}
        caption="{t.monitor.usage} · {t.monitor.cores(m.cores.length)}"
        tone={level(m.cpu, 85, 95)}
      />
      <div class="split">
        <Stat label={t.monitor.splitUser} value={formatPct(m.cpuSplit.user)} />
        <Stat label={t.monitor.splitKernel} value={formatPct(m.cpuSplit.system)} />
        <Stat label={t.monitor.splitIowait} value={formatPct(m.cpuSplit.iowait)} />
        <Stat label={t.monitor.splitSteal} value={formatPct(m.cpuSplit.steal)} />
      </div>
    </div>
    <TimeChart
      samples={data.history}
      {range}
      end={m.at}
      {rangeLabel}
      label={t.monitor.cpu}
      series={[{ label: t.monitor.cpu, color: chartColors[0], value: (s) => s.cpu }]}
      max={100}
      format={formatPct}
      height={150}
    />
  </Card>

  {#if m.cores.length > 0}
    <Card title={t.monitor.perCore} icon="grid">
      <div class="cores">
        {#each m.cores as pct, i (i)}
          <div class="core">
            <span class="name">{i}</span>
            <Meter {pct} label={t.monitor.core(i)} />
            <span class="pct">{formatPct(pct)}</span>
          </div>
        {/each}
      </div>
    </Card>
  {/if}

  <Card title={t.monitor.load} icon="monitor">
    <div class="split">
      <Stat label={t.monitor.load1} value={loadFormat(m.load1)} color={chartColors[0]} />
      <Stat label={t.monitor.load5} value={loadFormat(m.load5)} />
      <Stat label={t.monitor.load15} value={loadFormat(m.load15)} />
    </div>
    <TimeChart
      samples={data.history}
      {range}
      end={m.at}
      {rangeLabel}
      label="{t.monitor.load} {t.monitor.load1}"
      series={[{ label: t.monitor.load1, color: chartColors[0], value: (s) => s.load1 }]}
      floor={Math.max(1, m.cores.length)}
      format={loadFormat}
      height={110}
    />
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

  .cores {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
    gap: 8px 18px;
  }

  .core {
    display: grid;
    grid-template-columns: 22px 1fr 40px;
    align-items: center;
    gap: 8px;
    font-size: var(--text-xs);
    font-variant-numeric: tabular-nums;
  }

  .name {
    color: var(--fg-subtle);
    text-align: right;
  }

  .pct {
    text-align: right;
    color: var(--fg-muted);
  }
</style>
