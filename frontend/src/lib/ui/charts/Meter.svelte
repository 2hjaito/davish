<script lang="ts">
  import { level } from './scale'

  interface Props {
    /** 0 to 100. */
    pct: number
    label: string
    warn?: number
    danger?: number
  }

  let { pct, label, warn = 85, danger = 95 }: Props = $props()

  const clamped = $derived(Math.min(100, Math.max(0, pct)))
</script>

<div
  class="meter {level(pct, warn, danger)}"
  role="meter"
  aria-label={label}
  aria-valuemin={0}
  aria-valuemax={100}
  aria-valuenow={Math.round(clamped)}
>
  <span style:width="{clamped}%"></span>
</div>

<style>
  .meter {
    height: 6px;
    border-radius: 3px;
    background: var(--fill-track);
    overflow: hidden;
  }

  span {
    display: block;
    height: 100%;
    border-radius: 3px;
    background: var(--accent);
    transition: width var(--dur-slow) var(--ease-out);
  }

  .warn span {
    background: var(--warn);
  }

  .danger span {
    background: var(--danger);
  }
</style>
