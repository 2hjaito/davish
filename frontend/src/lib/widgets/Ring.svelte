<script lang="ts">
  import type { Snippet } from 'svelte'

  import { ringDash } from './gauge'

  interface Props {
    /** How full the ring is, 0 to 1. */
    value: number
    size?: number
    stroke?: number
    /** A CSS color. */
    color?: string
    /** What sits in the middle. */
    children?: Snippet
  }

  let { value, size = 56, stroke = 7, color = 'var(--widget-blue)', children }: Props = $props()

  const r = $derived((size - stroke) / 2)
  const c = $derived(size / 2)
</script>

<span class="ring" style:width="{size}px" style:height="{size}px">
  <svg viewBox="0 0 {size} {size}" aria-hidden="true">
    <circle cx={c} cy={c} {r} fill="none" stroke="var(--widget-track)" stroke-width={stroke} />
    {#if value > 0.005}
      <circle
        cx={c}
        cy={c}
        {r}
        fill="none"
        stroke={color}
        stroke-width={stroke}
        stroke-linecap="round"
        stroke-dasharray={ringDash(value, r)}
        transform="rotate(-90 {c} {c})"
      />
    {/if}
  </svg>
  {#if children}<span class="center">{@render children()}</span>{/if}
</span>

<style>
  .ring {
    position: relative;
    display: inline-flex;
    flex: none;
  }

  svg {
    width: 100%;
    height: 100%;
  }

  .center {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: var(--text-xs);
    font-variant-numeric: tabular-nums;
  }
</style>
