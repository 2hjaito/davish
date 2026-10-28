<script lang="ts">
  import type { AppId } from '../../apps/types'
  import { t } from '../i18n/vi'
  import { ms } from '../motion'
  import AppIcon from '../ui/AppIcon.svelte'
  import { approach, magnify, MAX_LIFT, MAX_SCALE } from './magnify'

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
    height?: number
  }

  let { apps, isOpen, onlaunch, height = $bindable(0) }: Props = $props()

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
    return (tile?.offsetWidth ?? 0) + gap
  }

  /** Aims the tiles at a viewport x; null relaxes them all back to 1. */
  function aim(clientX: number | null) {
    if (clientX === null) {
      targets = apps.map(() => 1)
    } else {
      // The dock stays centered while it widens, so its center is a stable origin.
      const r = dock.getBoundingClientRect()
      targets = magnify(clientX - (r.left + r.width / 2), apps.length, step())
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
    aim(r.left + r.width / 2)
  }

  $effect(() => () => cancelAnimationFrame(frame))

  function launch(app: Tile) {
    if (!onlaunch(app.id)) return
    bouncing = null
    // Restart the animation on the next frame if the same tile is clicked again.
    requestAnimationFrame(() => (bouncing = app.id))
  }
</script>

<nav
  class="dock glass"
  aria-label={t.dock.label}
  bind:this={dock}
  bind:offsetHeight={height}
  onpointermove={(e) => aim(e.clientX)}
  onpointerleave={() => aim(null)}
  onfocusin={aimAtTile}
  onfocusout={() => aim(null)}
>
  {#each apps as app, i (app.id)}
    {@const scale = scales[i] ?? 1}
    <div
      class="slot"
      style:--scale={scale}
      style:--lift="{(MAX_LIFT * (scale - 1)) / (MAX_SCALE - 1)}px"
    >
      <span class="tooltip glass glass-strong" aria-hidden="true">{app.label}</span>
      <button
        class="tile"
        class:bounce={bouncing === app.id}
        data-dock-app={app.id}
        data-own-tooltip
        aria-label={app.label}
        onclick={() => launch(app)}
        onanimationend={() => (bouncing = null)}
      >
        <AppIcon id={app.id} />
      </button>
      <span class="dot" class:on={isOpen(app.id)}></span>
    </div>
  {/each}
</nav>

<style>
  .dock {
    position: absolute;
    bottom: var(--dock-bottom);
    left: 50%;
    transform: translateX(-50%);
    z-index: 900;
    display: flex;
    align-items: flex-end;
    /* The gap lives in the slots' padding so the hover area has no dead zones. */
    padding: 9px calc(12px - var(--dock-gap) / 2) 7px;
    border-radius: var(--dock-radius);
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
    transform: translateY(calc(-1 * var(--lift))) scale(var(--scale));
    transition: filter var(--dur-fast) var(--ease-out);
  }

  .tile :global(svg) {
    display: block;
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
      translate: 0 -18px;
      animation-timing-function: cubic-bezier(0.6, 0, 0.8, 0.3);
    }
    55% {
      translate: 0 0;
      animation-timing-function: cubic-bezier(0.2, 0.7, 0.4, 1);
    }
    75% {
      translate: 0 -7px;
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
