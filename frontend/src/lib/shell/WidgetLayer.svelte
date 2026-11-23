<script lang="ts">
  import type { AppId } from '../../apps/types'
  import { t } from '../i18n'
  import { servers } from '../stores/servers.svelte'
  import { ui } from '../stores/ui.svelte'
  import { widgets } from '../stores/widgets.svelte'
  import Icon from '../ui/Icon.svelte'
  import {
    cellsOf,
    footprintAt,
    footprintBox,
    layoutWidgets,
    placeWidget,
    wantFor,
    type WidgetItem,
    type WidgetSize,
  } from '../widgets/layout'
  import { widgetDefs } from '../widgets/registry'
  import { wm } from '../wm/windowManager.svelte'
  import { CELL_H, CELL_W, cellAt, gridOf } from './iconGrid'

  interface Props {
    /** A click on a widget opens the app it shows. */
    onopen: (appId: AppId) => void
  }

  let { onopen }: Props = $props()

  /** The pointer must move this far before a press is a drag. */
  const DRAG_START = 5

  const serverID = $derived(servers.activeId)
  const grid = $derived(gridOf(wm.area))
  const placed = $derived(layoutWidgets(widgets.of(serverID), grid))
  const editing = $derived(ui.widgetEdit)

  /** The Done button sits under the widgets, at their right edge: notifications cover the top. */
  const doneAt = $derived.by(() => {
    const boxes = placed.flatMap((p) => (p.at ? [footprintBox(p.at, wm.area)] : []))
    if (boxes.length === 0) return null
    return {
      top: Math.max(...boxes.map((b) => b.y + b.h)) + 10,
      right: Math.max(...boxes.map((b) => b.x + b.w)),
    }
  })

  let press = $state<{
    id: string
    x: number
    y: number
    /** Where the widget's box was when pressed. */
    boxX: number
    boxY: number
  } | null>(null)
  let drag = $state<{ id: string; dx: number; dy: number } | null>(null)

  /** Set for the moment after a drag ends: the click that follows it is not a click on the widget. */
  let justDragged = false

  function onPointerDown(ev: PointerEvent, item: WidgetItem, box: { x: number; y: number }) {
    // A widget drags as a desktop icon does, in or out of edit mode: a press that
    // moves a few pixels is a drag, one that does not is a click.
    if (ev.button !== 0) return
    press = { id: item.id, x: ev.clientX, y: ev.clientY, boxX: box.x, boxY: box.y }
  }

  // The move and the release are heard on the window, not on the widget: the
  // pointer leaves a widget as soon as it is dragged, and capturing it is not
  // something every input path honours.
  function onPointerMove(ev: PointerEvent) {
    if (!press) return
    const dx = ev.clientX - press.x
    const dy = ev.clientY - press.y
    if (!drag && Math.hypot(dx, dy) < DRAG_START) return
    drag = { id: press.id, dx, dy }
  }

  function onPointerUp() {
    const item =
      press && drag?.id === press.id
        ? widgets.of(serverID).find((w) => w.id === press!.id)
        : undefined
    if (item && press && drag) {
      const cell = dropCell(press.boxX, press.boxY, drag.dx, drag.dy)
      widgets.move(serverID, item.id, wantFor(cell, item.size, grid), grid)
      justDragged = true
      setTimeout(() => (justDragged = false), 0)
    }
    drag = null
    press = null
  }

  function cancelDrag() {
    drag = null
    press = null
  }

  /** The cell a dragged widget's top left is over, for dropping it and for the outline. */
  function dropCell(boxX: number, boxY: number, dx: number, dy: number) {
    // The cell under the middle of the first cell of the dragged box.
    return cellAt(boxX + dx + CELL_W / 2, boxY + dy + CELL_H / 2, wm.area, grid)
  }

  /** Where the widget being dragged would land, drawn as an outline. */
  const ghost = $derived.by(() => {
    if (!drag || !press) return null
    const item = widgets.of(serverID).find((w) => w.id === drag!.id)
    if (!item) return null
    const cell = dropCell(press.boxX, press.boxY, drag.dx, drag.dy)
    const others = layoutWidgets(
      widgets.of(serverID).filter((w) => w.id !== item.id),
      grid,
    )
    const taken = new Set(
      others.flatMap((p) => (p.at ? cellsOf(p.at).map((c) => `${c.col}:${c.row}`) : [])),
    )
    const spot = placeWidget(wantFor(cell, item.size, grid), item.size, taken, grid)
    return spot ? footprintBox(footprintAt(spot.col, spot.row, item.size, grid), wm.area) : null
  })

  const SIZES: WidgetSize[] = ['S', 'M']
  const sizeLabel = (s: WidgetSize) => (s === 'S' ? t.widgets.small : t.widgets.medium)

  function onKeyDown(ev: KeyboardEvent) {
    if (editing && ev.key === 'Escape') ui.widgetEdit = false
  }
</script>

<svelte:window
  onkeydown={onKeyDown}
  onpointermove={onPointerMove}
  onpointerup={onPointerUp}
  onpointercancel={cancelDrag}
/>

{#if placed.length > 0}
  <div class="surface" role="group" aria-label={t.widgets.edit}>
    {#each placed as p (p.item.id)}
      {#if p.at}
        {@const box = footprintBox(p.at, wm.area)}
        {@const def = widgetDefs[p.item.kind]}
        {@const moving = drag?.id === p.item.id}
        <section
          class="widget"
          role="group"
          aria-label={def.label()}
          class:editing
          class:moving
          style:left="{box.x}px"
          style:top="{box.y}px"
          style:width="{box.w}px"
          style:height="{box.h}px"
          style:transform={moving && drag ? `translate(${drag.dx}px, ${drag.dy}px)` : undefined}
          onpointerdown={(ev) => onPointerDown(ev, p.item, box)}
        >
          {#if def.opens && !editing}
            <button class="body" onclick={() => !justDragged && def.opens && onopen(def.opens)}>
              {#await def.load() then W}<W.default {serverID} size={p.item.size} />{/await}
            </button>
          {:else}
            <div class="body">
              {#await def.load() then W}<W.default {serverID} size={p.item.size} />{/await}
            </div>
          {/if}
          {#if editing}
            <div class="tools" role="toolbar" aria-label={def.label()}>
              {#if def.sizes.length > 1}
                <span class="sizes" role="radiogroup" aria-label={t.widgets.resize}>
                  {#each SIZES.filter((s) => def.sizes.includes(s)) as s (s)}
                    <button
                      class="size"
                      role="radio"
                      aria-checked={p.item.size === s}
                      aria-label={sizeLabel(s)}
                      title={sizeLabel(s)}
                      onpointerdown={(ev) => ev.stopPropagation()}
                      onclick={() => widgets.resize(serverID, p.item.id, s, grid)}
                    >
                      {s}
                    </button>
                  {/each}
                </span>
              {/if}
              <button
                class="remove"
                aria-label={t.widgets.remove}
                title={t.widgets.remove}
                onpointerdown={(ev) => ev.stopPropagation()}
                onclick={() => widgets.remove(serverID, p.item.id)}
              >
                <Icon name="close" size={11} strokeWidth={2.5} />
              </button>
            </div>
          {/if}
        </section>
      {/if}
    {/each}
    {#if ghost}
      <div
        class="ghost"
        style:left="{ghost.x}px"
        style:top="{ghost.y}px"
        style:width="{ghost.w}px"
        style:height="{ghost.h}px"
      ></div>
    {/if}
    {#if editing && doneAt}
      <button
        class="finish"
        style:top="{doneAt.top}px"
        style:left="{doneAt.right}px"
        onclick={() => (ui.widgetEdit = false)}
      >
        {t.widgets.finish}
      </button>
    {/if}
  </div>
{/if}

<style>
  .surface {
    position: absolute;
    inset: 0;
    z-index: 0;
    pointer-events: none;
  }

  /*
   * Only the place and the shape: the face (a dark card, tinted by the state)
   * is the widget's own, with no blur of its own since widgets are always on
   * screen.
   */
  .widget {
    position: absolute;
    box-sizing: border-box;
    border-radius: var(--widget-radius);
    pointer-events: auto;
  }

  .widget {
    touch-action: none;
    user-select: none;
  }

  .widget.editing {
    outline: 2px dashed var(--accent);
    outline-offset: 3px;
    cursor: grab;
  }

  /* Where the widget being dragged will land. */
  .ghost {
    position: absolute;
    box-sizing: border-box;
    border: 2px dashed var(--accent);
    border-radius: var(--widget-radius);
    background: var(--accent-soft);
    pointer-events: none;
  }

  .widget.moving {
    z-index: 1;
    opacity: 0.8;
    cursor: grabbing;
  }

  .body {
    display: block;
    width: 100%;
    height: 100%;
    margin: 0;
    padding: 0;
    border: 0;
    background: none;
    color: inherit;
    font: inherit;
    text-align: left;
  }

  button.body {
    cursor: pointer;
  }

  .tools {
    position: absolute;
    top: -12px;
    right: 10px;
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .sizes {
    display: flex;
    border: 1px solid var(--widget-edge);
    border-radius: 999px;
    background: var(--widget-base);
  }

  .size,
  .remove {
    display: flex;
    align-items: center;
    justify-content: center;
    min-width: 20px;
    height: 20px;
    padding: 0 6px;
    border: 0;
    border-radius: 999px;
    background: none;
    color: var(--fg-on-dark-muted);
    font-size: var(--text-xxs);
    font-weight: 600;
    cursor: pointer;
  }

  .size[aria-checked='true'] {
    background: var(--accent);
    color: var(--fg-on-accent);
  }

  .remove {
    padding: 0;
    border: 1px solid var(--widget-edge);
    background: var(--widget-base);
  }

  .remove:hover {
    color: var(--danger-on-dark);
  }

  .finish {
    position: absolute;
    translate: -100% 0;
    padding: 5px 16px;
    border: 0;
    border-radius: 999px;
    background: var(--accent);
    color: var(--fg-on-accent);
    font: inherit;
    font-size: var(--text-sm);
    font-weight: 600;
    pointer-events: auto;
    cursor: pointer;
  }
</style>
