<script lang="ts">
  import { formatBytes, formatDuration, formatPct, formatRate } from '../../lib/format'
  import { t } from '../../lib/i18n/vi'
  import { level } from '../../lib/ui/charts/scale'
  import TimeChart from '../../lib/ui/charts/TimeChart.svelte'
  import Card from './Card.svelte'
  import Hero from './Hero.svelte'
  import Stat from './Stat.svelte'
  import { chartColors, memPct, type Section, type ViewProps } from './types'

  interface Props extends ViewProps {
    open: (s: Section) => void
  }

  let { data, range, rangeLabel, open }: Props = $props()

  const m = $derived(data.latest)
  const mem = $derived(memPct(m.memUsed, m.memTotal))
  const root = $derived(m.mounts[0])
  const chart = $derived({
    samples: data.history,
    range,
    end: m.at,
    rangeLabel,
    height: 56,
    compact: true,
  })
</script>

<div class="grid">
  <Card title={t.monitor.cpu} icon="cpu" onopen={() => open('cpu')}>
    <Hero
      value={formatPct(m.cpu)}
      caption={t.monitor.cores(m.cores.length)}
      tone={level(m.cpu, 85, 95)}
    />
    <TimeChart
      {...chart}
      label={t.monitor.cpu}
      series={[{ label: t.monitor.cpu, color: chartColors[0], value: (s) => s.cpu }]}
      max={100}
      format={formatPct}
    />
  </Card>

  <Card title={t.monitor.memory} icon="memory" onopen={() => open('memory')}>
    <Hero
      value={formatBytes(m.memUsed)}
      caption={t.monitor.ofTotal(formatBytes(m.memTotal))}
      tone={level(mem, 90, 95)}
    />
    <TimeChart
      {...chart}
      label={t.monitor.memoryUsed}
      series={[{ label: t.monitor.used, color: chartColors[0], value: (s) => s.memUsed }]}
      max={m.memTotal}
      format={formatBytes}
    />
  </Card>

  <Card title={t.monitor.disk} icon="disk" onopen={() => open('disk')}>
    <Hero
      value={formatPct(m.diskPct)}
      caption="{t.monitor.rootDisk} · {t.monitor.ofTotal(formatBytes(root?.size ?? 0))}"
      tone={level(m.diskPct, 85, 95)}
    />
    <TimeChart
      {...chart}
      label={t.monitor.throughput}
      series={[
        { label: t.monitor.read, color: chartColors[0], value: (s) => s.readBps },
        { label: t.monitor.write, color: chartColors[1], value: (s) => s.writeBps },
      ]}
      binary
      floor={64 * 1024}
      format={formatRate}
    />
  </Card>

  <Card title={t.monitor.network} icon="network" onopen={() => open('network')}>
    <div class="pair">
      <Stat label={t.monitor.rx} value={formatRate(m.rxBps)} color={chartColors[0]} />
      <Stat label={t.monitor.tx} value={formatRate(m.txBps)} color={chartColors[1]} />
    </div>
    <TimeChart
      {...chart}
      label={t.monitor.traffic}
      series={[
        { label: t.monitor.rx, color: chartColors[0], value: (s) => s.rxBps },
        { label: t.monitor.tx, color: chartColors[1], value: (s) => s.txBps },
      ]}
      binary
      floor={64 * 1024}
      format={formatRate}
    />
  </Card>
</div>

<div class="facts">
  <Stat
    label={t.monitor.load}
    value="{m.load1.toFixed(2)} · {m.load5.toFixed(2)} · {m.load15.toFixed(2)}"
  />
  <Stat label={t.monitor.uptime} value={formatDuration(m.uptime, t.units)} />
  <Stat
    label={t.monitor.swap}
    value={m.swapTotal > 0
      ? `${formatBytes(m.swapUsed)} / ${formatBytes(m.swapTotal)}`
      : t.monitor.unknown}
  />
  <Stat
    label={t.monitor.gpu}
    value={m.gpus[0] && m.gpus[0].util >= 0 ? formatPct(m.gpus[0].util) : t.monitor.unknown}
  />
</div>

<style>
  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
    gap: 12px;
  }

  .pair {
    display: flex;
    gap: 24px;
    /* Same height as a Hero, so the charts of a row line up. */
    min-height: 51px;
    align-items: center;
  }

  .facts {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
    gap: 12px;
    margin-top: 14px;
    padding: 12px 14px;
    border-radius: var(--card-radius);
    border: 1px solid var(--divider-soft);
  }
</style>
