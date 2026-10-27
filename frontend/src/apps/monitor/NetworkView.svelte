<script lang="ts">
  import { formatBytes, formatRate } from '../../lib/format'
  import { t } from '../../lib/i18n/vi'
  import TimeChart from '../../lib/ui/charts/TimeChart.svelte'
  import Card from './Card.svelte'
  import Stat from './Stat.svelte'
  import { chartColors, type ViewProps } from './types'

  let { data, range, rangeLabel }: ViewProps = $props()

  const m = $derived(data.latest)
</script>

<div class="stack">
  <Card title={t.monitor.traffic} icon="network">
    <div class="split">
      <Stat label={t.monitor.rx} value={formatRate(m.rxBps)} color={chartColors[0]} />
      <Stat label={t.monitor.tx} value={formatRate(m.txBps)} color={chartColors[1]} />
    </div>
    <TimeChart
      samples={data.history}
      {range}
      end={m.at}
      {rangeLabel}
      label={t.monitor.traffic}
      series={[
        { label: t.monitor.rx, color: chartColors[0], value: (s) => s.rxBps },
        { label: t.monitor.tx, color: chartColors[1], value: (s) => s.txBps },
      ]}
      binary
      floor={64 * 1024}
      format={formatRate}
      height={150}
    />
  </Card>

  <Card title={t.monitor.interfaces} icon="globe">
    {#if m.ifaces.length === 0}
      <p class="muted">{t.monitor.noIfaces}</p>
    {:else}
      <div class="scroll">
        <table class="mon-table">
          <thead>
            <tr>
              <th>{t.monitor.ifaceName}</th>
              <th class="num">{t.monitor.rx}</th>
              <th class="num">{t.monitor.tx}</th>
              <th class="num">{t.monitor.rxTotal}</th>
              <th class="num">{t.monitor.txTotal}</th>
            </tr>
          </thead>
          <tbody>
            {#each m.ifaces as n (n.name)}
              <tr>
                <td>{n.name}</td>
                <td class="num">{formatRate(n.rxBps)}</td>
                <td class="num">{formatRate(n.txBps)}</td>
                <td class="num muted">{formatBytes(n.rxTotal)}</td>
                <td class="num muted">{formatBytes(n.txTotal)}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
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
  }

  .scroll {
    overflow-x: auto;
    margin: 0 -6px;
  }

  .muted {
    margin: 0;
    font-size: var(--text-sm);
    color: var(--fg-muted);
  }
</style>
