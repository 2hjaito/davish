<script lang="ts">
  interface Part {
    kind: string
    count: number
  }

  interface Props {
    parts: readonly Part[]
    /** A CSS color for each kind; a kind without one is drawn dim. */
    colors: Record<string, string>
    height?: number
  }

  let { parts, colors, height = 8 }: Props = $props()

  const shown = $derived(parts.filter((p) => p.count > 0))
</script>

<!-- One segment per kind, as wide as its share; a small one still shows. -->
<div class="bar" style:height="{height}px" aria-hidden="true">
  {#each shown as p (p.kind)}
    <span style:flex-grow={p.count} style:background={colors[p.kind] ?? 'var(--widget-track)'}
    ></span>
  {/each}
</div>

<style>
  .bar {
    display: flex;
    gap: 2px;
    overflow: hidden;
    border-radius: 999px;
    background: var(--widget-track);
  }

  .bar span {
    flex-basis: 0;
    min-width: 4px;
    border-radius: 999px;
  }
</style>
