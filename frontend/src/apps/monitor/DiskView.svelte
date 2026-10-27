<script lang="ts">
  import { formatBytes, formatPct, formatRate } from '../../lib/format'
  import { t } from '../../lib/i18n/vi'
  import Meter from '../../lib/ui/charts/Meter.svelte'
  import TimeChart from '../../lib/ui/charts/TimeChart.svelte'
  import Card from './Card.svelte'
  import Stat from './Stat.svelte'
  import { chartColors, type ViewProps } from './types'

  let { data, range, rangeLabel }: ViewProps = $props()

  const m = $derived(data.latest)
</script>

<div class="stack">
  <Card title={t.monitor.throughput} icon="disk">
    <div class="split">
      <Stat label={t.monitor.read} value={formatRate(m.readBps)} color={chartColors[0]} />
      <Stat label={t.monitor.write} value={formatRate(m.writeBps)} color={chartColors[1]} />
      {#if m.disks.length > 1}
        {#each m.disks as d (d.name)}
          <Stat label={d.name} value="↓ {formatRate(d.readBps)}  ↑ {formatRate(d.writeBps)}" />
        {/each}
      {/if}
    </div>
    <TimeChart
      samples={data.history}
      {range}
      end={m.at}
      {rangeLabel}
      label={t.monitor.throughput}
      series={[
        { label: t.monitor.read, color: chartColors[0], value: (s) => s.readBps },
        { label: t.monitor.write, color: chartColors[1], value: (s) => s.writeBps },
      ]}
      binary
      floor={64 * 1024}
      format={formatRate}
      height={150}
    />
  </Card>

  <Card title={t.monitor.mounts} icon="folder">
    <div class="scroll">
      <table class="mon-table">
        <thead>
          <tr>
            <th>{t.monitor.mountPath}</th>
            <th>{t.monitor.device}</th>
            <th class="num">{t.monitor.size}</th>
            <th class="num">{t.monitor.used}</th>
            <th class="num">{t.monitor.avail}</th>
            <th class="usage">{t.monitor.usage}</th>
          </tr>
        </thead>
        <tbody>
          {#each m.mounts as mt (mt.path)}
            <tr>
              <td>{mt.path}</td>
              <td class="muted">{mt.device}</td>
              <td class="num">{formatBytes(mt.size)}</td>
              <td class="num">{formatBytes(mt.used)}</td>
              <td class="num">{formatBytes(mt.avail)}</td>
              <td class="usage">
                <span class="bar">
                  <Meter pct={mt.pct} label="{mt.path} {formatPct(mt.pct)}" />
                  <span class="num">{formatPct(mt.pct)}</span>
                </span>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  </Card>
</div>

<style>
  .stack {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .split {
    display: flex;
    gap: 22px;
    flex-wrap: wrap;
  }

  .scroll {
    overflow-x: auto;
    margin: 0 -6px;
  }

  .usage {
    width: 150px;
  }

  .bar {
    display: grid;
    grid-template-columns: 1fr 38px;
    align-items: center;
    gap: 8px;
  }
</style>
