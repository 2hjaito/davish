<script lang="ts">
  interface Props {
    /** A formatted reading, "23%" or "5,1 GB"; the unit is drawn smaller. */
    value: string
    caption?: string
    tone?: '' | 'warn' | 'danger'
  }

  let { value, caption = '', tone = '' }: Props = $props()

  const parts = $derived(/^([\d.,]+)\s*(.*)$/.exec(value))
  const num = $derived(parts?.[1] ?? value)
  const unit = $derived(parts?.[2] ?? '')
</script>

<div class="hero">
  <span class="value {tone}">{num}<small>{unit}</small></span>
  {#if caption}<span class="caption">{caption}</span>{/if}
</div>

<style>
  .hero {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  /* Proportional figures: tabular digits look loose at this size. */
  .value {
    font-size: 30px;
    font-weight: 600;
    line-height: 1.1;
    letter-spacing: -0.01em;
  }

  small {
    margin-left: 3px;
    font-size: var(--text-lg);
    font-weight: 500;
    color: var(--fg-muted);
  }

  .warn {
    color: var(--warn);
  }

  .danger {
    color: var(--danger);
  }

  .caption {
    font-size: var(--text-xs);
    color: var(--fg-muted);
  }
</style>
