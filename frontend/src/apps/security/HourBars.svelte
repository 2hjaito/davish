<script lang="ts">
  import type { HourCount } from '../../lib/api/security'
  import { t } from '../../lib/i18n'
  import { hourLabel, shares } from './overview'

  let { hours }: { hours: readonly HourCount[] } = $props()

  const s = t.security

  const heights = $derived(shares(hours.map((h) => h.count)))
  const peak = $derived(Math.max(0, ...hours.map((h) => h.count)))
  // The hour under the pointer or picked with the arrow keys.
  let at = $state<number | null>(null)
  const shown = $derived(at === null ? null : hours[at])

  function move(e: PointerEvent) {
    const box = (e.currentTarget as HTMLElement).getBoundingClientRect()
    const i = Math.floor(((e.clientX - box.left) / box.width) * hours.length)
    at = hours.length ? Math.min(hours.length - 1, Math.max(0, i)) : null
  }

  function key(e: KeyboardEvent) {
    const step = e.key === 'ArrowLeft' ? -1 : e.key === 'ArrowRight' ? 1 : 0
    if (!step || !hours.length) return
    e.preventDefault()
    at = Math.min(hours.length - 1, Math.max(0, (at ?? hours.length) + step))
  }
</script>

<div class="head">
  <h3>{s.hourly}</h3>
  <span class="tip" aria-live="polite"
    >{shown ? s.hourTip(hourLabel(shown.hour), shown.count) : ''}</span
  >
</div>
<!-- svelte-ignore a11y_no_noninteractive_tabindex, a11y_no_noninteractive_element_interactions -->
<div
  class="bars"
  role="application"
  aria-label={s.hourly}
  tabindex="0"
  onkeydown={key}
  onblur={() => (at = null)}
  onpointermove={move}
  onpointerleave={() => (at = null)}
>
  {#each hours as h, i (h.hour)}
    <div class="slot" class:on={at === i}>
      <span style:height="{heights[i]}%"></span>
    </div>
  {/each}
</div>
<div class="axis">
  <span>{hourLabel(hours[0]?.hour ?? '')}</span>
  <span>{s.peak(peak)}</span>
  <span>{hourLabel(hours.at(-1)?.hour ?? '')}</span>
</div>

<style>
  .head {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 12px;
  }

  h3 {
    margin: 0 0 8px;
    font-size: var(--text-xs);
    font-weight: 600;
    text-transform: uppercase;
    color: var(--fg-label);
  }

  .tip {
    font-size: var(--text-xs);
    color: var(--fg-muted);
    font-variant-numeric: tabular-nums;
  }

  .bars {
    height: 110px;
    display: flex;
    align-items: stretch;
    gap: 1px;
    border-bottom: 1px solid var(--chart-grid);
    outline-offset: 2px;
  }

  .slot {
    flex: 1;
    min-width: 0;
    display: flex;
    align-items: flex-end;
    border-radius: 2px;
  }

  .slot.on {
    background: var(--chart-grid);
  }

  .slot span {
    width: 100%;
    border-radius: 2px 2px 0 0;
    background: var(--chart-1);
  }

  .axis {
    display: flex;
    justify-content: space-between;
    margin-top: 4px;
    font-size: 10.5px;
    color: var(--fg-subtle);
    font-variant-numeric: tabular-nums;
  }
</style>
