<script lang="ts">
  import type { Unit } from '../../lib/api/services'
  import { t } from '../../lib/i18n/vi'
  import { health, shortName } from './units'

  interface Props {
    units: readonly Unit[]
    selected: string | null
    onselect: (name: string) => void
  }

  let { units, selected, onselect }: Props = $props()

  // Must match .row's height in CSS, for paging and scrolling into view.
  const ROW_H = 30

  let scroller = $state<HTMLDivElement>()
  let viewport = $state(0)
  const listId = $props.id()
  const selectedIndex = $derived(units.findIndex((u) => u.name === selected))

  function move(i: number) {
    const u = units[Math.max(0, Math.min(units.length - 1, i))]
    if (!u) return
    onselect(u.name)
    document.getElementById(`${listId}-${units.indexOf(u)}`)?.scrollIntoView({ block: 'nearest' })
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
      End: () => move(units.length - 1),
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
  aria-label={t.services.search}
  aria-activedescendant={selectedIndex >= 0 ? `${listId}-${selectedIndex}` : undefined}
  bind:this={scroller}
  bind:clientHeight={viewport}
  {onkeydown}
>
  {#each units as u, i (u.name)}
    <button
      id="{listId}-{i}"
      class="row"
      class:odd={i % 2 === 1}
      class:selected={u.name === selected}
      role="option"
      aria-selected={u.name === selected}
      tabindex="-1"
      title={u.description}
      onclick={() => {
        scroller?.focus({ preventScroll: true })
        onselect(u.name)
      }}
    >
      <span class="dot {health(u.active)}" aria-hidden="true"></span>
      <span class="name">{shortName(u.name)}</span>
      <span class="sub">{t.services.states[u.active] ?? u.active}</span>
    </button>
  {:else}
    <p class="none">{t.services.noMatch}</p>
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
