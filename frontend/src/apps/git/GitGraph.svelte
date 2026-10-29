<script lang="ts">
  import type { Commit, Ref } from '../../lib/api/git'
  import { t } from '../../lib/i18n'
  import Icon from '../../lib/ui/Icon.svelte'
  import { layout, type Segment } from './graph'

  // The history as a commit graph, styled and driven like StatusList.
  interface Props {
    commits: readonly Commit[]
    /** False while searching: the matches are not a continuous history. */
    graph: boolean
    selected: string | null
    label: string
    empty: string
    onselect: (hash: string) => void
  }

  let { commits, graph, selected, label, empty, onselect }: Props = $props()

  // Must match .row's height in CSS: lines meet across rows.
  const ROW_H = 30
  const LANE = 14
  /** Lanes past this are cut off, for a repository with many branches. */
  const MAX_LANES = 8
  const LANE_COLORS = 7

  const rows = $derived(graph ? layout(commits) : [])
  const lanes = $derived(Math.min(MAX_LANES, Math.max(1, ...rows.map((r) => r.width))))

  let scroller = $state<HTMLDivElement>()
  let viewport = $state(0)
  const listId = $props.id()
  const selectedIndex = $derived(commits.findIndex((c) => c.hash === selected))

  const x = (lane: number) => LANE / 2 + lane * LANE
  const color = (lane: number) => `var(--graph-${(lane % LANE_COLORS) + 1})`

  function path(s: Segment): string {
    const mid = ROW_H / 2
    const [y1, y2] = s.half === 'top' ? [0, mid] : s.half === 'bottom' ? [mid, ROW_H] : [0, ROW_H]
    const [x1, x2] = [x(s.from), x(s.to)]
    if (x1 === x2) return `M${x1} ${y1}V${y2}`
    const c = (y1 + y2) / 2
    return `M${x1} ${y1}C${x1} ${c} ${x2} ${c} ${x2} ${y2}`
  }

  const isHead = (c: Commit) => c.refs.some((r) => r.kind === 'head')

  function refTitle(r: Ref): string {
    return t.git.refKinds[r.kind] ?? r.kind
  }

  function move(i: number) {
    const at = Math.max(0, Math.min(commits.length - 1, i))
    const c = commits[at]
    if (!c) return
    onselect(c.hash)
    document.getElementById(`${listId}-${at}`)?.scrollIntoView({ block: 'nearest' })
  }

  function onkeydown(ev: KeyboardEvent) {
    if (ev.metaKey || ev.ctrlKey || ev.altKey) return
    const page = Math.max(1, Math.floor(viewport / ROW_H) - 1)
    const keys: Record<string, () => void> = {
      ArrowDown: () => move(selectedIndex + 1),
      ArrowUp: () => move(selectedIndex < 0 ? 0 : selectedIndex - 1),
      PageDown: () => move(selectedIndex + page),
      PageUp: () => move(selectedIndex - page),
      Home: () => move(0),
      End: () => move(commits.length - 1),
    }
    const action = keys[ev.key]
    if (!action) return
    ev.preventDefault()
    action()
  }
</script>

<div
  class="scroller"
  role="listbox"
  tabindex="0"
  aria-label={label}
  aria-activedescendant={selectedIndex >= 0 ? `${listId}-${selectedIndex}` : undefined}
  bind:this={scroller}
  bind:clientHeight={viewport}
  {onkeydown}
>
  {#each commits as c, i (c.hash)}
    {@const row = rows[i]}
    <button
      id="{listId}-{i}"
      class="row"
      class:odd={i % 2 === 1}
      class:selected={c.hash === selected}
      role="option"
      aria-selected={c.hash === selected}
      tabindex="-1"
      title="{c.short} · {c.author} · {new Date(c.time * 1000).toLocaleString('vi-VN')}"
      onclick={() => {
        scroller?.focus({ preventScroll: true })
        onselect(c.hash)
      }}
    >
      {#if row}
        <svg
          class="graph"
          width={lanes * LANE}
          height={ROW_H}
          viewBox="0 0 {lanes * LANE} {ROW_H}"
          aria-hidden="true"
        >
          {#each row.segments as s, n (n)}
            {#if s.from < MAX_LANES && s.to < MAX_LANES}
              <path d={path(s)} style:stroke={color(s.color)} />
            {/if}
          {/each}
          {#if row.lane < MAX_LANES}
            <circle
              class="node"
              class:head={isHead(c)}
              class:merge={c.parents.length > 1}
              cx={x(row.lane)}
              cy={ROW_H / 2}
              r={isHead(c) ? 5 : 4}
              style:--lane={color(row.lane)}
            />
          {/if}
        </svg>
      {/if}
      {#each c.refs as r (r.kind + r.name)}
        <span class="ref {r.kind}" title={refTitle(r)}>
          {#if r.kind !== 'tag'}<Icon name="gitBranch" size={10} strokeWidth={2.4} />{/if}
          {r.name}
        </span>
      {/each}
      <span class="name">{c.subject}</span>
      <span class="sub">{new Date(c.time * 1000).toLocaleDateString('vi-VN')}</span>
    </button>
  {:else}
    <p class="none">{empty}</p>
  {/each}
</div>

<style>
  .scroller {
    flex: 1;
    min-height: 0;
    padding: 4px 8px 6px;
    overflow-y: auto;
    scrollbar-gutter: stable;
    outline: none;
  }

  .row {
    display: flex;
    align-items: center;
    gap: 6px;
    width: 100%;
    height: 30px;
    padding: 0 10px 0 4px;
    border: 0;
    border-radius: var(--control-radius);
    background: none;
    text-align: left;
    font-size: var(--text-md);
  }

  .row.odd {
    background: var(--row-stripe);
  }

  .row:hover {
    background: var(--row-hover);
  }

  .row.selected {
    background: var(--selection-inactive);
  }

  .scroller:focus-within .row.selected {
    background: var(--accent);
  }

  .graph {
    flex-shrink: 0;
    overflow: visible;
  }

  .graph path {
    fill: none;
    stroke-width: 2;
    stroke-linecap: round;
  }

  .node {
    fill: var(--lane);
    stroke: var(--content-bg);
    stroke-width: 1.5;
  }

  /* A merge is hollow; the checked out commit gets a ring. */
  .node.merge {
    fill: var(--content-bg);
    stroke: var(--lane);
    stroke-width: 2;
  }

  .node.head {
    fill: var(--content-bg);
    stroke: var(--lane);
    stroke-width: 3;
  }

  .ref {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    flex-shrink: 0;
    max-width: 30%;
    padding: 1px 6px;
    border: 1px solid transparent;
    border-radius: 999px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: var(--text-xxs);
    font-weight: 600;
  }

  .ref.head {
    background: var(--accent);
    color: var(--fg-on-accent);
  }

  .ref.local {
    background: var(--accent-soft);
    color: var(--accent);
  }

  .ref.remote {
    background: var(--fill-soft);
    color: var(--fg-muted);
  }

  .ref.tag {
    border-color: var(--graph-6);
    color: var(--graph-6);
  }

  .name {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .sub {
    font-size: var(--text-xs);
    color: var(--fg-muted);
    white-space: nowrap;
    font-variant-numeric: tabular-nums;
  }

  .scroller:focus-within .selected :is(.name, .sub) {
    color: var(--fg-on-accent);
  }

  /* On the accent row every badge turns white, so none disappears. */
  .scroller:focus-within .selected .ref {
    border: 1px solid var(--fg-on-accent);
    background: none;
    color: var(--fg-on-accent);
  }

  .scroller:focus-within .selected .ref.head {
    background: var(--fg-on-accent);
    color: var(--accent);
  }

  .none {
    margin: 24px 0;
    text-align: center;
    color: var(--fg-subtle);
  }
</style>
