<script lang="ts" module>
  /** One row: a status dot, a name and a short note on the right. */
  export interface StatusItem {
    key: string
    name: string
    health: 'ok' | 'busy' | 'failed' | 'off'
    note: string
    /** Tooltip of the row. */
    title?: string
  }
</script>

<script lang="ts">
  // The selectable list of Services and Docker, styled like Files.
  interface Props {
    items: readonly StatusItem[]
    selected: string | null
    label: string
    empty: string
    onselect: (key: string) => void
  }

  let { items, selected, label, empty, onselect }: Props = $props()

  // Must match .row's height in CSS, for paging and scrolling into view.
  const ROW_H = 30

  let scroller = $state<HTMLDivElement>()
  let viewport = $state(0)
  const listId = $props.id()
  const selectedIndex = $derived(items.findIndex((u) => u.key === selected))

  function move(i: number) {
    const at = Math.max(0, Math.min(items.length - 1, i))
    const u = items[at]
    if (!u) return
    onselect(u.key)
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
      End: () => move(items.length - 1),
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
  {#each items as u, i (u.key)}
    <button
      id="{listId}-{i}"
      class="row"
      class:odd={i % 2 === 1}
      class:selected={u.key === selected}
      role="option"
      aria-selected={u.key === selected}
      tabindex="-1"
      title={u.title}
      onclick={() => {
        scroller?.focus({ preventScroll: true })
        onselect(u.key)
      }}
    >
      <span class="dot {u.health}" aria-hidden="true"></span>
      <span class="name">{u.name}</span>
      <span class="sub">{u.note}</span>
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
    gap: 8px;
    width: 100%;
    height: 30px;
    padding: 0 10px;
    border: 0;
    border-radius: var(--control-radius);
    background: none;
    text-align: left;
    font-size: var(--text-md);
  }

  /* Zebra stripes, like Finder's list view. */
  .row.odd {
    background: var(--row-stripe);
  }

  .row:hover {
    background: var(--row-hover);
  }

  /* As in Files: solid accent while the list has focus, gray otherwise. */
  .row.selected {
    background: var(--selection-inactive);
  }

  .scroller:focus-within .row.selected {
    background: var(--accent);
  }

  .dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .dot.ok {
    background: var(--ok);
  }

  .dot.busy {
    background: var(--warn);
  }

  .dot.failed {
    background: var(--danger);
  }

  .dot.off {
    background: var(--offline);
  }

  .name {
    flex: 1;
    min-width: 0;
    font-weight: 500;
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

  /* The dot keeps its meaning on the accent row with a white ring. */
  .scroller:focus-within .selected .dot {
    box-shadow: 0 0 0 1.5px var(--fg-on-accent);
  }

  .none {
    margin: 24px 0;
    text-align: center;
    color: var(--fg-subtle);
  }
</style>
