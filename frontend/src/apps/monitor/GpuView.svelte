<script lang="ts">
  import type { Gpu } from '../../lib/api/events'
  import { formatBytes, formatPct } from '../../lib/format'
  import { t } from '../../lib/i18n/vi'
  import Meter from '../../lib/ui/charts/Meter.svelte'
  import TimeChart from '../../lib/ui/charts/TimeChart.svelte'
  import Card from './Card.svelte'
  import Empty from './Empty.svelte'
  import Stat from './Stat.svelte'
  import { chartColors, memPct, type ViewProps } from './types'

  let { data, range, rangeLabel }: ViewProps = $props()

  const m = $derived(data.latest)
  const known = (v: number, show: (v: number) => string) => (v >= 0 ? show(v) : t.monitor.unknown)
  const vramPct = (g: Gpu) => (g.memTotal > 0 ? memPct(g.memUsed, g.memTotal) : 0)
  const gpuAt = (s: { gpus: Gpu[] }, index: number) => s.gpus.find((g) => g.index === index)
</script>

{#if m.gpus.length === 0}
  <Empty icon="gpu" title={t.monitor.gpuNone} hint={t.monitor.gpuNoneHint} />
{:else}
  <div class="stack">
    {#each m.gpus as g (g.index)}
      <Card title={g.name || `${t.monitor.gpu} ${g.index}`} icon="gpu">
        {#snippet aside()}<span>{t.monitor.gpu} {g.index}</span>{/snippet}
        <div class="stats">
          <Stat label={t.monitor.gpuUtil} value={known(g.util, formatPct)} color={chartColors[0]} />
          <Stat
            label={t.monitor.vram}
            value="{known(g.memUsed, formatBytes)} / {known(g.memTotal, formatBytes)}"
            color={chartColors[1]}
          />
          <Stat label={t.monitor.temp} value={known(g.temp, (v) => `${Math.round(v)} °C`)} />
          <Stat label={t.monitor.power} value={known(g.power, (v) => `${Math.round(v)} W`)} />
        </div>
        <div class="meters">
          <Meter pct={Math.max(0, g.util)} label={t.monitor.gpuUtil} />
          <Meter pct={vramPct(g)} label={t.monitor.vram} warn={90} danger={97} />
        </div>
        <TimeChart
          samples={data.history}
          {range}
          end={m.at}
          {rangeLabel}
          label="{g.name} {t.monitor.gpuUtil}"
          series={[
            {
              label: t.monitor.gpuUtil,
              color: chartColors[0],
              value: (s) => Math.max(0, gpuAt(s, g.index)?.util ?? 0),
            },
            {
              label: t.monitor.vram,
              color: chartColors[1],
              value: (s) => {
                const h = gpuAt(s, g.index)
                return h ? vramPct(h) : 0
              },
            },
          ]}
          max={100}
          format={formatPct}
          height={140}
        />
      </Card>
    {/each}
  </div>
{/if}

<style>
  .stack {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .stats {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(110px, 1fr));
    gap: 12px;
  }

  .meters {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 18px;
  }
</style>
