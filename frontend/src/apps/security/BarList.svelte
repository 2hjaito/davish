<script lang="ts">
  import type { Snippet } from 'svelte'
  import { shares } from './overview'

  interface Row {
    key: string
    count: number
  }

  interface Props {
    title: string
    rows: readonly Row[]
    /** The label of a row, with any tags after it. */
    label: Snippet<[Row]>
  }

  let { title, rows, label }: Props = $props()

  const widths = $derived(shares(rows.map((r) => r.count)))
</script>

<h3>{title}</h3>
<ul>
  {#each rows as r, i (r.key)}
    <li>
      <span class="label">{@render label(r)}</span>
      <span class="track"><span style:width="{widths[i]}%"></span></span>
      <span class="num">{r.count}</span>
    </li>
  {/each}
</ul>

<style>
  h3 {
    margin: 0 0 8px;
    font-size: var(--text-xs);
    font-weight: 600;
    text-transform: uppercase;
    color: var(--fg-label);
  }

  ul {
    list-style: none;
    margin: 0;
    padding: 0;
  }

  li {
    display: grid;
    grid-template-columns: minmax(0, 1.3fr) minmax(0, 1fr) 3.5em;
    align-items: center;
    gap: 10px;
    height: 24px;
    font-size: var(--text-sm);
  }

  .label {
    display: flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
    overflow: hidden;
    white-space: nowrap;
  }

  .track {
    height: 6px;
    border-radius: 3px;
    background: var(--fill-track);
    overflow: hidden;
  }

  .track span {
    display: block;
    height: 100%;
    border-radius: 3px;
    background: var(--chart-1);
  }

  .num {
    text-align: right;
    font-variant-numeric: tabular-nums;
    color: var(--fg-muted);
  }
</style>
