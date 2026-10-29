<script lang="ts" module>
  export type DragPhase = 'start' | 'move' | 'end' | 'cancel'
</script>

<script lang="ts">
  import type { Entry } from '../../lib/api/files'
  import { formatBytes } from '../../lib/format'
  import { t } from '../../lib/i18n'
  import Icon from '../../lib/ui/Icon.svelte'
  import {
    formatModTime,
    scrollToRow,
    SORT_KEYS,
    type SortKey,
    type SortOrder,
    visibleRows,
  } from './listing'

  interface Props {
    entries: readonly Entry[]
    order: SortOrder
    selected: string | null
    /** When the listing was fetched; dates read "today" relative to it. */
    now: Date
    onsort: (key: SortKey) => void
    onselect: (path: string) => void
    onopen: (entry: Entry) => void
    /**
     * A row dragged out of place, for the drag menu. Positions are in
     * viewport pixels; start gives where the press began.
     */
    ondrag?: (entry: Entry, phase: DragPhase, x: number, y: number) => void
  }

  let { entries, order, selected, now, onsort, onselect, onopen, ondrag }: Props = $props()

  // Must match .row's height in CSS; the virtual window is computed from it.
  const ROW_H = 30

  let scroller = $state<HTMLDivElement>()
  let scrollTop = $state(0)
  let viewport = $state(0)

  const win = $derived(visibleRows(scrollTop, viewport, ROW_H, entries.length))
  const rows = $derived(entries.slice(win.start, win.end))
  const selectedIndex = $derived(entries.findIndex((e) => e.path === selected))
  const listId = $props.id()

  const labels: Record<SortKey, string> = {
    name: t.files.name,
    size: t.files.size,
    modified: t.files.modified,
  }

  /** Where the list is scrolled, for the caller to come back to later. */
  export function getScroll(): number {
    return scroller?.scrollTop ?? 0
  }

  /**
   * Scrolls to top (or keeps the position when null), then brings path into
   * view if it is listed. Call after the entries are rendered.
   */
  export function restore(top: number | null, path: string | null) {
    if (!scroller) return
    if (top !== null) scroller.scrollTop = top
    scrollTop = scroller.scrollTop
    const i = path === null ? -1 : entries.findIndex((e) => e.path === path)
    if (i >= 0) reveal(i)
  }

  function reveal(index: number) {
    if (!scroller) return
    const top = scrollToRow(index, scroller.scrollTop, scroller.clientHeight, ROW_H)
    if (top !== null) scroller.scrollTop = top
  }

  function move(index: number) {
    const e = entries[Math.max(0, Math.min(entries.length - 1, index))]
    if (!e) return
    onselect(e.path)
    reveal(entries.indexOf(e))
  }

  // Keys go to the scroller, never to a row: rows scrolled out of the
  // window are removed, and focus would fall to the page with them.
  // A press becomes a drag only after it moves this far, so clicks and
  // double clicks keep working.
  const DRAG_START = 12

  let press: { entry: Entry; x: number; y: number; pointer: number } | null = null
  let dragging = false
  // The click that ends a drag must not also select a row.
  let swallowClick = false

  function onRowPointerDown(ev: PointerEvent, e: Entry) {
    if (!ondrag || ev.button !== 0 || ev.metaKey || ev.ctrlKey || ev.shiftKey || ev.altKey) return
    press = { entry: e, x: ev.clientX, y: ev.clientY, pointer: ev.pointerId }
  }

  function onpointermove(ev: PointerEvent) {
    if (!press || !ondrag || ev.pointerId !== press.pointer) return
    if (!dragging) {
      if (Math.hypot(ev.clientX - press.x, ev.clientY - press.y) < DRAG_START) return
      dragging = true
      // Keep receiving moves wherever the pointer goes, rows included.
      scroller?.setPointerCapture(ev.pointerId)
      select(press.entry.path)
      ondrag(press.entry, 'start', press.x, press.y)
    }
    ondrag(press.entry, 'move', ev.clientX, ev.clientY)
  }

  function endDrag(ev: PointerEvent, phase: 'end' | 'cancel') {
    if (press && dragging && ondrag) {
      ondrag(press.entry, phase, ev.clientX, ev.clientY)
      // The click follows pointerup in the same task, if it comes at all.
      swallowClick = phase === 'end'
      setTimeout(() => (swallowClick = false))
    }
    press = null
    dragging = false
  }

  function onWindowKeydown(ev: KeyboardEvent) {
    if (ev.key !== 'Escape' || !dragging || !press || !ondrag) return
    ev.preventDefault()
    ondrag(press.entry, 'cancel', press.x, press.y)
    press = null
    dragging = false
  }

  function onRowClick(path: string) {
    if (swallowClick) {
      swallowClick = false
      return
    }
    select(path)
  }

  function select(path: string) {
    scroller?.focus({ preventScroll: true })
    onselect(path)
  }

  function onkeydown(ev: KeyboardEvent) {
    const page = Math.max(1, Math.floor(viewport / ROW_H) - 1)
    const keys: Record<string, () => void> = {
      ArrowDown: () => move(selectedIndex + 1),
      ArrowUp: () => move(selectedIndex < 0 ? 0 : selectedIndex - 1),
      PageDown: () => move(selectedIndex + page),
      PageUp: () => move(selectedIndex - page),
      Home: () => move(0),
      End: () => move(entries.length - 1),
      Enter: () => {
        const e = entries[selectedIndex]
        if (e) onopen(e)
      },
    }
    const action = ev.metaKey || ev.ctrlKey || ev.altKey ? undefined : keys[ev.key]
    if (!action) return
    ev.preventDefault()
    action()
  }

  function ariaSort(key: SortKey) {
    if (order.key !== key) return 'none'
    return order.desc ? 'descending' : 'ascending'
  }

  function icon(e: Entry) {
    if (e.dir) return 'folder'
    return e.kind === 'link' ? 'link' : 'file'
  }
</script>

<svelte:window onkeydown={onWindowKeydown} />

<div class="table">
  <div class="row head" role="row">
    {#each SORT_KEYS as key (key)}
      <button
        class="sort"
        class:num={key === 'size'}
        class:active={order.key === key}
        role="columnheader"
        aria-sort={ariaSort(key)}
        onclick={() => onsort(key)}
      >
        {labels[key]}
        {#if order.key === key}
          <Icon name={order.desc ? 'chevronDown' : 'arrowUp'} size={11} strokeWidth={2.4} />
        {/if}
      </button>
    {/each}
  </div>

  <div
    class="scroller"
    role="listbox"
    tabindex="0"
    aria-label={t.files.listLabel}
    aria-activedescendant={selectedIndex >= 0 ? `${listId}-${selectedIndex}` : undefined}
    bind:this={scroller}
    bind:clientHeight={viewport}
    onscroll={() => scroller && (scrollTop = scroller.scrollTop)}
    {onkeydown}
    {onpointermove}
    onpointerup={(ev) => endDrag(ev, 'end')}
    onpointercancel={(ev) => endDrag(ev, 'cancel')}
  >
    <div class="rows" style:height="{entries.length * ROW_H}px">
      {#each rows as e, i (e.path)}
        {@const index = win.start + i}
        <button
          id="{listId}-{index}"
          class="row"
          class:odd={index % 2 === 1}
          class:selected={e.path === selected}
          role="option"
          aria-selected={e.path === selected}
          tabindex="-1"
          style:translate="0 {index * ROW_H}px"
          onpointerdown={(ev) => onRowPointerDown(ev, e)}
          onclick={() => onRowClick(e.path)}
          ondblclick={() => onopen(e)}
          title={e.kind === 'link' && e.target ? `${e.name} → ${e.target}` : e.name}
        >
          <span class="name" class:dir={e.dir}>
            <Icon name={icon(e)} size={16} strokeWidth={e.dir ? 1.5 : 1.8} />
            <span class="label">{e.name}</span>
          </span>
          <span class="meta num">
            {#if e.dir}<span class="none">--</span>{:else}{formatBytes(e.size)}{/if}
          </span>
          <span class="meta">{formatModTime(e.modTime, now, t.files)}</span>
        </button>
      {/each}
    </div>
  </div>
</div>

<style>
  .table {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    font-size: var(--text-md);
  }

  .row {
    height: 30px;
    width: 100%;
    display: grid;
    grid-template-columns: minmax(0, 1fr) 76px 116px;
    gap: 14px;
    padding: 0 10px;
    border: 0;
    border-radius: var(--control-radius);
    background: transparent;
    align-items: center;
    text-align: left;
  }

  /* The header reserves the same scrollbar gutter as the list so the
     columns line up where scrollbars take space (Windows, Linux). */
  .head {
    flex-shrink: 0;
    height: 28px;
    padding: 0 18px;
    border-radius: 0;
    border-bottom: 1px solid var(--divider-soft);
    overflow-y: hidden;
    scrollbar-gutter: stable;
  }

  .sort {
    display: flex;
    align-items: center;
    gap: 3px;
    min-width: 0;
    padding: 0;
    border: 0;
    background: none;
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--fg-muted);
  }

  .sort.num {
    justify-content: flex-end;
  }

  .sort.active {
    color: var(--fg);
  }

  .scroller {
    flex: 1;
    min-height: 0;
    padding: 4px 8px 6px;
    overflow-y: auto;
    scrollbar-gutter: stable;
    outline: none;
  }

  .rows {
    position: relative;
  }

  .rows .row {
    position: absolute;
    top: 0;
    left: 0;
  }

  /* Zebra stripes, like Finder's list view, guide the eye along a row. */
  .row.odd {
    background: var(--row-stripe);
  }

  .rows .row:hover {
    background: var(--row-hover);
  }

  /* Like Finder: a solid accent row, clearly apart from the hover tint; it
     turns gray while the list does not have focus. */
  .rows .row.selected {
    background: var(--selection-inactive);
  }

  .scroller:focus-within .row.selected {
    background: var(--accent);
  }

  .name {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
    color: var(--file-stroke);
  }

  .name.dir {
    color: var(--folder-stroke);
  }

  .name.dir :global(svg *) {
    fill: var(--folder-fill);
  }

  .label {
    color: var(--fg);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .meta {
    color: var(--fg-muted);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  /* Like Finder: sizes line up on the right so their magnitudes compare at a glance. */
  .num {
    text-align: right;
    font-variant-numeric: tabular-nums;
  }

  /* Folders have no size. Plain figures keep the two dashes tight. */
  .none {
    font-variant-numeric: normal;
    color: var(--fg-subtle);
  }

  .scroller:focus-within .selected :is(.name, .label, .meta) {
    color: var(--fg-on-accent);
  }

  .scroller:focus-within .selected .name.dir :global(svg *) {
    fill: var(--fg-on-accent);
  }
</style>
