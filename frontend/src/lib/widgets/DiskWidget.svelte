<script lang="ts">
  import { formatBytes, formatPct } from '../format'
  import { t } from '../i18n'
  import { metrics } from '../stores/metrics.svelte'
  import { settings } from '../stores/settings.svelte'
  import Bar from './Bar.svelte'
  import { type DiskLevel, summarizeMounts } from './diskUsage'
  import type { WidgetProps } from './registry'
  import Ring from './Ring.svelte'
  import WidgetCard, { type Tone } from './WidgetCard.svelte'

  let { serverID, size }: WidgetProps = $props()

  const d = t.widgets.disk

  // Shares the collector with the menu bar and Monitor: one watch more, no command more.
  $effect(() => metrics.watch(serverID))

  const latest = $derived(metrics.latest(serverID))
  const s = $derived(latest ? summarizeMounts(latest.mounts, settings.alertDisk) : null)

  const TONE: Record<DiskLevel, Tone> = { '': 'neutral', warn: 'warn', danger: 'danger' }
  const COLOR: Record<DiskLevel, string> = {
    '': 'var(--widget-blue)',
    warn: 'var(--warn-on-dark)',
    danger: 'var(--danger-on-dark)',
  }

  const tone = $derived<Tone>(s ? TONE[s.level] : 'neutral')
  const state = $derived(
    s?.level === 'danger' ? d.almostFull : s?.level === 'warn' ? d.filling : '',
  )

  const NAMED = 5
  const rows = $derived(s ? s.rows.slice(0, NAMED) : [])
  const more = $derived(s ? s.rows.length - rows.length : 0)
  const top = $derived(s?.fullest ?? null)
</script>

<WidgetCard {tone}>
  {#if !s || !top}
    <p class="wait">{s ? d.none : t.widgets.waiting}</p>
  {:else if size === 'S'}
    <div class="s">
      <div class="text">
        <span class="label">{t.apps.disk} <span class="path">{top.path}</span></span>
        <span class="hero">{formatPct(top.pct)}</span>
        <span class="sub" title={d.of(formatBytes(top.used), formatBytes(top.size))}>
          {d.ofTotal(formatBytes(top.size))}
        </span>
      </div>
      <Ring value={top.pct / 100} size={60} stroke={7} color={COLOR[top.level]} />
    </div>
  {:else}
    <div class="m">
      <div class="head">
        <span class="label">{t.apps.disk}</span>
        {#if state}<span class="state {tone}">{state}</span>{/if}
      </div>
      <ul>
        {#each rows as r (r.path)}
          <li title={d.of(formatBytes(r.used), formatBytes(r.size))}>
            <div class="line">
              <span class="name">{r.path}</span>
              <span class="pct {r.level}">{formatPct(r.pct)}</span>
            </div>
            <Bar
              parts={[
                { kind: 'used', count: r.pct },
                { kind: 'free', count: Math.max(0, 100 - r.pct) },
              ]}
              colors={{ used: COLOR[r.level], free: 'var(--widget-track)' }}
              height={5}
            />
            <span class="sub">{d.of(formatBytes(r.used), formatBytes(r.size))}</span>
          </li>
        {/each}
        {#if more > 0}<li class="more">{d.more(more)}</li>{/if}
      </ul>
    </div>
  {/if}
</WidgetCard>

<style>
  .s {
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

  .label {
    overflow: hidden;
    color: var(--fg-on-dark-muted);
    font-size: var(--text-xxs);
    font-weight: 600;
    letter-spacing: 0.05em;
    text-overflow: ellipsis;
    text-transform: uppercase;
    white-space: nowrap;
  }

  .path {
    margin-left: 2px;
    letter-spacing: 0;
    text-transform: none;
  }

  .hero {
    color: var(--fg-on-dark);
    font-size: 30px;
    font-variant-numeric: tabular-nums;
    font-weight: 650;
    line-height: 1.1;
    white-space: nowrap;
  }

  .sub {
    overflow: hidden;
    color: var(--fg-on-dark-muted);
    font-size: var(--text-xs);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .m {
    display: flex;
    flex-direction: column;
    gap: 6px;
    height: 100%;
    min-height: 0;
  }

  .head {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 8px;
  }

  .state {
    font-size: var(--text-xs);
    font-weight: 600;
    white-space: nowrap;
  }

  .state.warn {
    color: var(--warn-on-dark);
  }

  .state.danger {
    color: var(--danger-on-dark);
  }

  ul {
    display: flex;
    flex-direction: column;
    gap: 7px;
    margin: 0;
    padding: 0;
    list-style: none;
  }

  li {
    display: flex;
    flex-direction: column;
    gap: 3px;
    min-width: 0;
  }

  li .sub {
    font-size: var(--text-xxs);
  }

  .line {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 8px;
    min-width: 0;
    font-size: var(--text-sm);
  }

  .name {
    overflow: hidden;
    color: var(--fg-on-dark);
    font-weight: 600;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .pct {
    color: var(--fg-on-dark);
    font-variant-numeric: tabular-nums;
    font-weight: 600;
  }

  .pct.warn {
    color: var(--warn-on-dark);
  }

  .pct.danger {
    color: var(--danger-on-dark);
  }

  li.more {
    color: var(--fg-on-dark-muted);
    font-size: var(--text-xs);
  }

  .wait {
    margin: auto 0;
    color: var(--fg-on-dark-muted);
    font-size: var(--text-xs);
  }
</style>
