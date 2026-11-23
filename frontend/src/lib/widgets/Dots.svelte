<script lang="ts">
  interface Props {
    /** The kind of each cell, left to right, row by row; 'empty' is the dim one. */
    cells: readonly string[]
    cols: number
    /** A CSS color for each kind; a kind without one is drawn dim. */
    colors: Record<string, string>
    size?: number
    gap?: number
  }

  let { cells, cols, colors, size = 8, gap = 3 }: Props = $props()

  const rows = $derived(Math.ceil(cells.length / cols))
  const w = $derived(cols * size + (cols - 1) * gap)
  const h = $derived(rows * size + Math.max(0, rows - 1) * gap)
</script>

<svg viewBox="0 0 {w} {h}" width={w} height={h} aria-hidden="true">
  {#each cells as kind, i (i)}
    <rect
      x={(i % cols) * (size + gap)}
      y={Math.floor(i / cols) * (size + gap)}
      width={size}
      height={size}
      rx={size * 0.32}
      fill={colors[kind] ?? 'var(--widget-track)'}
    />
  {/each}
</svg>

<style>
  svg {
    display: block;
    flex: none;
  }
</style>
