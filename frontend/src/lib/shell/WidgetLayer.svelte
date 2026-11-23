<script lang="ts">
  import type { AppId } from '../../apps/types'
  import { t } from '../i18n'
  import { servers } from '../stores/servers.svelte'
  import { ui } from '../stores/ui.svelte'
  import { widgets } from '../stores/widgets.svelte'
  import Icon from '../ui/Icon.svelte'
  import {
    footprintBox,
    layoutWidgets,
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

  let press = $state<{ id: string; x: number; y: number; pointer: number } | null>(null)
  let drag = $state<{ id: string; dx: number; dy: number } | null>(null)

  function onPointerDown(ev: PointerEvent, item: WidgetItem) {
    if (!editing || ev.button !== 0) return
    press = { id: item.id, x: ev.clientX, y: ev.clientY, pointer: ev.pointerId }
  }

  function onPointerMove(ev: PointerEvent) {
    if (!press || ev.pointerId !== press.pointer) return
    const dx = ev.clientX - press.x
    const dy = ev.clientY - press.y
    if (!drag && Math.hypot(dx, dy) < DRAG_START) return
    ;(ev.currentTarget as HTMLElement).setPointerCapture(ev.pointerId)
    drag = { id: press.id, dx, dy }
  }

  function onPointerUp(item: WidgetItem, boxX: number, boxY: number) {
    if (drag?.id === item.id) {
      // The cell under the middle of the first cell of the dragged box.
      const cell = cellAt(boxX + drag.dx + CELL_W / 2, boxY + drag.dy + CELL_H / 2, wm.area, grid)
      widgets.move(serverID, item.id, wantFor(cell, item.size, grid), grid)
    }
    drag = null
    press = null
  }

  const SIZES: WidgetSize[] = ['S', 'M']
  const sizeLabel = (s: WidgetSize) => (s === 'S' ? t.widgets.small : t.widgets.medium)

  function onKeyDown(ev: KeyboardEvent) {
    if (editing && ev.key === 'Escape') ui.widgetEdit = false
  }
</script>

<svelte:window onkeydown={onKeyDown} />

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
          onpointerdown={(ev) => onPointerDown(ev, p.item)}
          onpointermove={onPointerMove}
          onpointerup={() => onPointerUp(p.item, box.x, box.y)}
          onpointercancel={() => ((drag = null), (press = null))}
        >
          {#if def.opens && !editing}
            <button class="body" onclick={() => def.opens && onopen(def.opens)}>
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

  /* A flat fill and a hairline, with no blur of its own: widgets are always on screen. */
  .widget {
    position: absolute;
    box-sizing: border-box;
    padding: 10px 12px;
    border: 1px solid var(--divider);
    border-radius: var(--card-radius);
    background: var(--content-bg);
    color: var(--fg);
    pointer-events: auto;
  }

  .widget.editing {
    border-style: dashed;
    border-color: var(--accent);
    cursor: grab;
    touch-action: none;
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
    top: -11px;
    right: 8px;
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .sizes {
    display: flex;
    border: 1px solid var(--divider);
    border-radius: 999px;
    background: var(--content-bg);
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
    color: var(--fg-muted);
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
    border: 1px solid var(--divider);
    background: var(--content-bg);
  }

  .remove:hover {
    color: var(--danger);
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
