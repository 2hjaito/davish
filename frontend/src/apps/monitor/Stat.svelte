<script lang="ts">
  interface Props {
    label: string
    value: string
    /** Series color, for a stat that doubles as a chart legend. */
    color?: string
  }

  let { label, value, color }: Props = $props()
</script>

<div class="stat">
  <span class="label">
    {#if color}<span class="key" style:background={color}></span>{/if}
    {label}
  </span>
  <span class="value">{value}</span>
</div>

<style>
  .stat {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }

  .label {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: var(--text-xs);
    color: var(--fg-muted);
    white-space: nowrap;
  }

  .key {
    width: 8px;
    height: 8px;
    border-radius: 2px;
    flex-shrink: 0;
  }

  .value {
    font-size: var(--text-md);
    font-weight: 600;
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
</style>
