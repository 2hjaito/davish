<script lang="ts">
  import type { AppId } from '../../apps/types'
  import { t } from '../i18n'
  import { dockPins } from '../stores/dock.svelte'
  import { DOCK_SIZES, settings } from '../stores/settings.svelte'
  import { flip } from 'svelte/animate'

  import { ms } from '../motion'
  import AppIcon from '../ui/AppIcon.svelte'
  import { moveItem } from './dockItems'
  import { approach, magnify, MAX_LIFT, MAX_SCALE, slotAt, slotCenter } from './magnify'

  /** An app's tile, or Launchpad's. */
  interface Tile {
    id: AppId | 'launchpad'
    label: string
  }

  interface Props {
    apps: Tile[]
    isOpen: (appId: string) => boolean
    /** Opens the tile's app; false when nothing opened, so it does not bounce. */
    onlaunch: (id: Tile['id']) => boolean
    /** Closes every window of the app in this workspace. */
    onclose: (appId: AppId) => void
    height?: number
    width?: number
  }

  let {
    apps,
    isOpen,
    onlaunch,
    onclose,
    height = $bindable(0),
    width = $bindable(0),
  }: Props = $props()

  /** Which edge the Dock sits on; magnify and the menu follow its axis. */
  const pos = $derived(settings.dockPosition)
  const vertical = $derived(pos === 'left' || pos === 'right')

  /** Closest the right-click menu's center gets to a screen edge: half its width and a margin. */
  const MENU_EDGE = 100

  // The right-click menu, like the macOS Dock's: open, keep in Dock, close.
  // style places it beside the tile, on the screen's side of the Dock.
  let menu = $state<{ app: Tile & { id: AppId }; style: string } | null>(null)
  let menuEl = $state<HTMLElement>()

  function openMenu(e: MouseEvent, app: Tile) {
    e.preventDefault()
    if (app.id === 'launchpad') return
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect()
    const x = Math.min(Math.max(r.left + r.width / 2, MENU_EDGE), window.innerWidth - MENU_EDGE)
    const y = Math.min(Math.max(r.top + r.height / 2, MENU_EDGE), window.innerHeight - MENU_EDGE)
    const style = {
      bottom: `left: ${x}px; bottom: ${window.innerHeight - r.top + 10}px; translate: -50% 0`,
      top: `left: ${x}px; top: ${r.bottom + 10}px; translate: -50% 0`,
      left: `left: ${r.right + 10}px; top: ${y}px; translate: 0 -50%`,
      right: `right: ${window.innerWidth - r.left + 10}px; top: ${y}px; translate: 0 -50%`,
    }[pos]
    menu = { app: { ...app, id: app.id }, style }
    requestAnimationFrame(() =>
      menuEl?.querySelector<HTMLElement>('button:not(:disabled)')?.focus(),
    )
  }

  // The action runs first: the menu's markup, and what it read, goes
  // away with menu.
  function pick(action: () => void) {
    action()
    menu = null
  }

  function onWindowPointer(e: PointerEvent) {
    if (menu && menuEl && !menuEl.contains(e.target as Node)) menu = null
  }

  let bouncing = $state<string | null>(null)
  let dock: HTMLElement
  let scales = $state<number[]>([])
  let targets: number[] = []
  let frame = 0
  let last = 0

  /** Distance between resting tile centers: tile width plus gap. */
  function step(): number {
    const tile = dock.querySelector<HTMLElement>('.tile')
    const gap = parseFloat(getComputedStyle(dock).getPropertyValue('--dock-gap')) || 0
    return (vertical ? (tile?.offsetHeight ?? 0) : (tile?.offsetWidth ?? 0)) + gap
  }

  /** Aims the tiles at a point along the Dock's axis; null relaxes them all back to 1. */
  function aim(at: number | null) {
    if (at === null) {
      targets = apps.map(() => 1)
    } else {
      // The dock stays centered while it grows, so its center is a stable origin.
      const r = dock.getBoundingClientRect()
      const center = vertical ? r.top + r.height / 2 : r.left + r.width / 2
      targets = magnify(at - center, apps.length, step())
    }
    if (scales.length !== apps.length) scales = apps.map(() => 1)
    if (ms(1) === 0) {
      scales = targets
      return
    }
    if (!frame) {
      last = performance.now()
      frame = requestAnimationFrame(tick)
    }
  }

  function tick(now: number) {
    const dt = Math.min(now - last, 50)
    last = now
    scales = scales.map((s, i) => approach(s, targets[i] ?? 1, dt))
    frame = scales.some((s, i) => s !== (targets[i] ?? 1)) ? requestAnimationFrame(tick) : 0
  }

  function aimAtTile(e: FocusEvent) {
    const tile = e.target as HTMLElement
    if (!tile.matches(':focus-visible')) return
    const r = tile.getBoundingClientRect()
    aim(vertical ? r.top + r.height / 2 : r.left + r.width / 2)
  }

  $effect(() => () => cancelAnimationFrame(frame))

  // Reorder, as on macOS: drag a pinned tile along the Dock and the others
  // part to show where it lands; letting go saves the new order. The drag
  // stays among the pinned tiles, after Launchpad and before open apps
  // that are not kept. Escape puts it back.

  /** A press becomes a drag only after it moves this far, so clicks stay clicks. */
  const DRAG_START = 6
  let press: { id: AppId; from: number; at: number; min: number; max: number } | null = null
  /** The tile being dragged, the slot it would land in, and how far it is off that slot. */
  let drag = $state<{ id: AppId; to: number; shift: number } | null>(null)
  // Letting go of a drag also clicks the tile, even one Escape called off;
  // that click must not launch it.
  let dragged = false

  /** apps with the dragged tile where it would land. */
  function arranged(d: { id: AppId; to: number }): Tile[] {
    const from = apps.findIndex((a) => a.id === d.id)
    return from < 0 ? apps : moveItem(apps, from, d.to)
  }

  const tiles = $derived(drag ? arranged(drag) : apps)

  function along(e: PointerEvent): number {
    return vertical ? e.clientY : e.clientX
  }

  function onPress(e: PointerEvent, app: Tile, i: number) {
    press = null
    dragged = false
    if (e.button !== 0 || app.id === 'launchpad' || !dockPins.has(app.id)) return
    // Pinned tiles come first, right after Launchpad, in one run.
    const min = apps.findIndex((a) => dockPins.has(a.id))
    const max = min + apps.filter((a) => dockPins.has(a.id)).length - 1
    press = { id: app.id, from: i, at: along(e), min, max }
  }

  function onDragMove(e: PointerEvent) {
    if (!press) return
    // The release happened outside the window and never came: forget the press.
    if (!(e.buttons & 1)) {
      cancelDrag()
      return
    }
    const moved = along(e) - press.at
    if (!drag) {
      if (Math.abs(moved) < DRAG_START) return
      ;(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId)
      dragged = true
    }
    // The tile's center follows the pointer, between the first and last pinned slots.
    const n = apps.length
    const s = step()
    const center = Math.min(
      Math.max(slotCenter(press.from, n, s) + moved, slotCenter(press.min, n, s)),
      slotCenter(press.max, n, s),
    )
    const to = Math.min(Math.max(slotAt(center, n, s), press.min), press.max)
    drag = { id: press.id, to, shift: center - slotCenter(to, n, s) }
  }

  function onRelease() {
    const d = drag
    cancelDrag()
    // The click comes right after the release, in the same task.
    if (dragged) setTimeout(() => (dragged = false))
    // Dropped where it was, or it left the Dock (another server) on the way.
    if (!d || apps[d.to]?.id === d.id) return
    const order = arranged(d)
    if (order[d.to]?.id !== d.id) return
    // Saved as "before the tile now after it", so pins this server does not
    // show (apps of another system) keep their places.
    const next = order[d.to + 1]
    dockPins.move(d.id, next && dockPins.has(next.id) ? next.id : null)
  }

  function cancelDrag() {
    press = null
    drag = null
  }

  // Autohide: the dock rests below the screen's edge and comes up while the
  // pointer, keyboard focus, its menu or a drag is on it.
  const HIDE_DELAY = 300
  let shown = $state(false)
  let hideTimer = 0
  const hidden = $derived(settings.dockAutohide && !shown && menu === null && drag === null)

  function reveal() {
    clearTimeout(hideTimer)
    shown = true
  }

  function hideSoon() {
    clearTimeout(hideTimer)
    hideTimer = window.setTimeout(() => (shown = false), HIDE_DELAY)
  }

  $effect(() => () => clearTimeout(hideTimer))

  function onTileClick(app: Tile) {
    if (!dragged) launch(app)
  }

  function launch(app: Tile) {
    if (!onlaunch(app.id)) return
    bouncing = null
    // Restart the animation on the next frame if the same tile is clicked again.
    requestAnimationFrame(() => (bouncing = app.id))
  }
</script>

<svelte:window
  onpointerdown={onWindowPointer}
  onkeydown={(e) => {
    if (e.key !== 'Escape') return
    menu = null
    cancelDrag()
  }}
  onblur={() => {
    menu = null
    cancelDrag()
  }}
/>

{#if settings.dockAutohide}
  <div class="edge" data-pos={pos} aria-hidden="true" onpointerenter={reveal}></div>
{/if}

<nav
  class="dock glass"
  class:clear={settings.dockTransparent}
  class:menu-open={menu !== null}
  class:dragging={drag !== null}
  class:hidden
  data-pos={pos}
  aria-label={t.dock.label}
  style:--count={apps.length}
  style:--dock-size="{DOCK_SIZES[settings.dockSize]}px"
  bind:this={dock}
  bind:offsetHeight={height}
  bind:offsetWidth={width}
  onpointerenter={reveal}
  onpointermove={(e) => aim(vertical ? e.clientY : e.clientX)}
  onpointerleave={() => {
    aim(null)
    hideSoon()
  }}
  onfocusin={(e) => {
    reveal()
    aimAtTile(e)
  }}
  onfocusout={() => {
    aim(null)
    hideSoon()
  }}
>
  {#each tiles as app, i (app.id)}
    {@const scale = scales[i] ?? 1}
    {@const shift = drag?.id === app.id ? drag.shift : null}
    <div
      class="slot"
      class:moving={shift !== null}
      style:--scale={scale}
      style:--lift="{(MAX_LIFT * (scale - 1)) / (MAX_SCALE - 1)}px"
      animate:flip={{ duration: shift === null ? ms(180) : 0 }}
    >
      <span class="tooltip glass glass-strong" aria-hidden="true">{app.label}</span>
      <button
        class="tile"
        class:bounce={bouncing === app.id}
        data-dock-app={app.id}
        data-own-tooltip
        aria-label={app.label}
        style:translate={shift === null ? undefined : vertical ? `0 ${shift}px` : `${shift}px 0`}
        onclick={() => onTileClick(app)}
        oncontextmenu={(e) => openMenu(e, app)}
        onanimationend={() => (bouncing = null)}
        onpointerdown={(e) => onPress(e, app, i)}
        onpointermove={onDragMove}
        onpointerup={onRelease}
        onpointercancel={cancelDrag}
      >
        <AppIcon id={app.id} />
      </button>
      <span class="dot" class:on={isOpen(app.id)}></span>
    </div>
  {/each}
</nav>

{#if menu}
  {@const m = menu}
  {@const open = isOpen(m.app.id)}
  <div
    bind:this={menuEl}
    class="menu glass glass-strong"
    role="menu"
    aria-label={m.app.label}
    style={m.style}
  >
    <p class="name">{m.app.label}</p>
    <button role="menuitem" onclick={() => pick(() => launch(m.app))}
      >{open ? t.dock.show : t.dock.open}</button
    >
    <div class="divider"></div>
    <button
      role="menuitemcheckbox"
      aria-checked={dockPins.has(m.app.id)}
      onclick={() => pick(() => dockPins.toggle(m.app.id))}
    >
      <span class="check">{dockPins.has(m.app.id) ? '✓' : ''}</span>
      {t.dock.keep}
    </button>
    {#if open}
      <div class="divider"></div>
      <button role="menuitem" onclick={() => pick(() => onclose(m.app.id))}>{t.dock.close}</button>
    {/if}
  </div>
{/if}

<style>
  /* The label would show through under the menu, and trail a dragged tile. */
  .dock.menu-open .slot .tooltip,
  .dock.dragging .slot .tooltip {
    visibility: hidden;
  }

  .menu {
    position: fixed;
    z-index: 950;
    min-width: 180px;
    padding: 5px;
    border-radius: var(--menu-radius);
    background: var(--glass-bg-menu);
    font-size: var(--text-md);
  }

  .menu .name {
    margin: 2px 10px 4px;
    font-weight: 600;
  }

  .menu button {
    width: 100%;
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 4px 10px;
    border: 0;
    border-radius: var(--item-radius);
    background: transparent;
    text-align: left;
  }

  .menu button:hover,
  .menu button:focus-visible {
    background: var(--accent);
    color: var(--fg-on-accent);
    outline: none;
  }

  .menu .check {
    width: 14px;
  }

  .menu .divider {
    height: 1px;
    margin: 4px 8px;
    background: var(--divider);
  }

  /*
   * Tiles shrink so every one stays on screen, however many are pinned. The
   * 84px holds the dock's padding, a margin and what magnify adds at the edges.
   */
  .dock {
    --dock-tile: min(var(--dock-size), calc((100vw - 84px) / var(--count) - var(--dock-gap)));
    --dock-tile-radius: calc(var(--dock-tile) * 0.27);
    /* Which way a tile lifts and bounces: away from the Dock's edge. */
    --away-x: 0;
    --away-y: -1;
    position: absolute;
    z-index: 900;
    display: flex;
    /* The gap lives in the slots' padding so the hover area has no dead zones. */
    border-radius: var(--dock-radius);
    transition: translate var(--dur) var(--ease-out);
    /* See MenuBar: refreshes the blur behind the Dock after a wallpaper change. */
    opacity: var(--backdrop-nudge, 1);
  }

  .dock[data-pos='bottom'] {
    bottom: var(--dock-bottom);
    left: 50%;
    transform: translateX(-50%);
    align-items: flex-end;
    padding: 9px calc(12px - var(--dock-gap) / 2) 7px;
  }

  .dock[data-pos='top'] {
    --away-y: 1;
    top: calc(var(--menubar-h) + 8px);
    left: 50%;
    transform: translateX(-50%);
    align-items: flex-start;
    padding: 7px calc(12px - var(--dock-gap) / 2) 9px;
  }

  .dock[data-pos='left'],
  .dock[data-pos='right'] {
    --dock-tile: min(
      var(--dock-size),
      calc((100vh - var(--menubar-h) - 84px) / var(--count) - var(--dock-gap))
    );
    --away-y: 0;
    /* Centered in the space under the menu bar. */
    top: calc(50% + var(--menubar-h) / 2);
    transform: translateY(-50%);
    flex-direction: column;
  }

  .dock[data-pos='left'] {
    --away-x: 1;
    left: var(--dock-bottom);
    align-items: flex-start;
    padding: calc(12px - var(--dock-gap) / 2) 9px calc(12px - var(--dock-gap) / 2) 7px;
  }

  .dock[data-pos='right'] {
    --away-x: -1;
    right: var(--dock-bottom);
    align-items: flex-end;
    padding: calc(12px - var(--dock-gap) / 2) 7px calc(12px - var(--dock-gap) / 2) 9px;
  }

  /* Past its edge, shadow and all. */
  .dock[data-pos='bottom'].hidden {
    translate: 0 calc(100% + var(--dock-bottom) + 12px);
  }

  .dock[data-pos='top'].hidden {
    translate: 0 calc(-100% - var(--menubar-h) - 20px);
  }

  .dock[data-pos='left'].hidden {
    translate: calc(-100% - var(--dock-bottom) - 12px) 0;
  }

  .dock[data-pos='right'].hidden {
    translate: calc(100% + var(--dock-bottom) + 12px) 0;
  }

  /* No glass: the icons float over the wallpaper, their own shadows kept. */
  .dock.glass.clear {
    background: none;
    border-color: transparent;
    box-shadow: none;
    -webkit-backdrop-filter: none;
    backdrop-filter: none;
  }

  /* Under the hidden dock: reaching its screen edge brings it back. */
  .edge {
    position: fixed;
    z-index: 899;
  }

  .edge[data-pos='bottom'] {
    left: 0;
    right: 0;
    bottom: 0;
    height: 4px;
  }

  .edge[data-pos='top'] {
    left: 0;
    right: 0;
    top: var(--menubar-h);
    height: 4px;
  }

  .edge[data-pos='left'],
  .edge[data-pos='right'] {
    top: var(--menubar-h);
    bottom: 0;
    width: 4px;
  }

  .edge[data-pos='left'] {
    left: 0;
  }

  .edge[data-pos='right'] {
    right: 0;
  }

  /*
   * Magnify: --scale/--lift are driven per frame from the pointer position (see
   * magnify.ts). The tile scales up from its bottom edge and the slot's margin
   * reserves the extra width, so neighbours are pushed aside (and the dock
   * widens) instead of being overlapped.
   */
  .slot {
    --scale: 1;
    --lift: 0px;
    position: relative;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    padding: 0 calc(var(--dock-gap) / 2);
    margin: 0 calc((var(--scale) - 1) * var(--dock-tile) / 2);
  }

  /* The dragged tile passes over its neighbours, not under them. */
  .slot.moving {
    z-index: 1;
  }

  /* The running dot stays on the screen-edge side of its tile. */
  [data-pos='top'] .slot {
    flex-direction: column-reverse;
  }

  [data-pos='left'] .slot,
  [data-pos='right'] .slot {
    flex-direction: row-reverse;
    padding: calc(var(--dock-gap) / 2) 0;
    margin: calc((var(--scale) - 1) * var(--dock-tile) / 2) 0;
  }

  [data-pos='right'] .slot {
    flex-direction: row;
  }

  /*
   * The artwork (AppIcon) fills the tile; the tile adds what every icon shares:
   * the squircle, a soft top sheen, a hairline bevel and the drop shadow.
   */
  .tile {
    position: relative;
    width: var(--dock-tile);
    height: var(--dock-tile);
    border: 0;
    padding: 0;
    border-radius: var(--dock-tile-radius);
    display: block;
    overflow: hidden;
    background: none;
    box-shadow: var(--tile-shadow);
    transform-origin: bottom center;
    transform: translate(calc(var(--away-x) * var(--lift)), calc(var(--away-y) * var(--lift)))
      scale(var(--scale));
    transition: filter var(--dur-fast) var(--ease-out);
  }

  [data-pos='top'] .tile {
    transform-origin: top center;
  }

  [data-pos='left'] .tile {
    transform-origin: center left;
  }

  [data-pos='right'] .tile {
    transform-origin: center right;
  }

  .tile :global(svg) {
    display: block;
    width: 100%;
    height: 100%;
  }

  .tile::after {
    content: '';
    position: absolute;
    inset: 0;
    border-radius: inherit;
    background: var(--tile-sheen);
    box-shadow: var(--tile-edge);
    pointer-events: none;
  }

  .tile:active {
    filter: brightness(0.82);
  }

  /* Launch bounce uses `translate` so it composes with the hover transform. */
  .bounce {
    animation: bounce calc(var(--dur-slow) * 2.7) linear;
  }

  /* Decelerate on the way up, accelerate on the way down, like a dropped ball. */
  @keyframes bounce {
    0% {
      translate: 0 0;
      animation-timing-function: cubic-bezier(0.2, 0.7, 0.4, 1);
    }
    30% {
      translate: calc(var(--away-x) * 18px) calc(var(--away-y) * 18px);
      animation-timing-function: cubic-bezier(0.6, 0, 0.8, 0.3);
    }
    55% {
      translate: 0 0;
      animation-timing-function: cubic-bezier(0.2, 0.7, 0.4, 1);
    }
    75% {
      translate: calc(var(--away-x) * 7px) calc(var(--away-y) * 7px);
      animation-timing-function: cubic-bezier(0.6, 0, 0.8, 0.3);
    }
    100% {
      translate: 0 0;
    }
  }

  .tooltip {
    position: absolute;
    /* Clears the hovered tile: lift (8px) + growth (0.2 × 56 ≈ 11px) + breathing room. */
    bottom: calc(100% + 27px);
    padding: 3px 10px;
    border-radius: var(--control-radius);
    font-size: var(--text-sm);
    white-space: nowrap;
    pointer-events: none;
    opacity: 0;
    translate: 0 4px;
    transition:
      opacity var(--dur-fast) var(--ease-out),
      translate var(--dur-fast) var(--ease-out);
  }

  /* The label sits on the side away from the screen edge. */
  [data-pos='top'] .tooltip {
    bottom: auto;
    top: calc(100% + 27px);
    translate: 0 -4px;
  }

  [data-pos='left'] .tooltip,
  [data-pos='right'] .tooltip {
    bottom: auto;
    top: 50%;
    translate: 0 -50%;
  }

  [data-pos='left'] .tooltip {
    left: calc(100% + 27px);
  }

  [data-pos='right'] .tooltip {
    right: calc(100% + 27px);
  }

  [data-pos='left'] .slot:hover .tooltip,
  [data-pos='right'] .slot:hover .tooltip,
  [data-pos='left'] .slot:has(.tile:focus-visible) .tooltip,
  [data-pos='right'] .slot:has(.tile:focus-visible) .tooltip {
    translate: 0 -50%;
  }

  .slot:hover .tooltip,
  .slot:has(.tile:focus-visible) .tooltip {
    opacity: 1;
    translate: 0 0;
    transition-delay: 150ms;
  }

  .dot {
    width: 5px;
    height: 5px;
    border-radius: 50%;
    background: var(--dock-dot);
    opacity: 0;
    transform: scale(0.4);
    transition:
      opacity var(--dur) var(--ease-out),
      transform var(--dur) var(--ease-out);
  }

  .dot.on {
    opacity: 1;
    transform: none;
  }
</style>
