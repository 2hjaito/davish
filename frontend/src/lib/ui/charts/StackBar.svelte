<script lang="ts">
  import type { Segment } from './scale'

  interface Props {
    segments: Segment[]
    /** The whole bar; what the segments leave is the empty track. */
    total: number
    label: string
  }

  let { segments, total, label }: Props = $props()

  const pct = (v: number) => (total > 0 ? (Math.max(0, v) / total) * 100 : 0)
</script>

<div class="stack" role="img" aria-label={label}>
  {#each segments as s (s.label)}
    {#if s.value > 0}
      <span style:width="{pct(s.value)}%" style:background={s.color} title={s.label}></span>
    {/if}
  {/each}
</div>

<style>
  /* Segments are parted by a 2px gap of the surface, never by a border. */
  .stack {
    display: flex;
    gap: 2px;
    height: 10px;
    border-radius: 5px;
    background: var(--fill-track);
    overflow: hidden;
  }

  span {
    display: block;
    height: 100%;
    transition: width var(--dur-slow) var(--ease-out);
  }
</style>
