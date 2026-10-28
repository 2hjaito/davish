<script lang="ts">
  import { type Snippet, untrack } from 'svelte'

  import { t } from '../i18n/vi'
  import { dockTile, GENIE_MS, ms, windowZoom } from '../motion'
  import { setWindowContext } from './context'
  import { type Edge, type Rect, resizeRect, snapZone, zoneRect } from './geometry'
  import { genieFrame } from './genie'
  import { type WindowState, wm } from './windowManager.svelte'

  interface Props {
    win: WindowState
    focused: boolean
    theme: 'light' | 'dark'
    children: Snippet
  }

  let { win, focused, theme, children }: Props = $props()

  const frame = $derived(wm.frame(win))
  const edges: Edge[] = ['n', 's', 'e', 'w', 'ne', 'nw', 'se', 'sw']
  /** Pointer travel before a title-bar press counts as a drag. */
  const DRAG_SLOP = 3

  let el = $state<HTMLElement>()

  // Minimize and restore pour the window into its dock tile and back out
  // (the genie); the .minimized class hides it once the animation is over.
  let wasMinimized = untrack(() => win.minimized)
  $effect(() => {
    const minimized = win.minimized
    if (minimized === wasMinimized) return
    wasMinimized = minimized
    untrack(() => genie(minimized))
  })

  function genie(minimizing: boolean) {
    const tile = dockTile(win.appId)
    const duration = ms(GENIE_MS)
    if (!el || !tile || duration === 0) return
    const box = { x: frame.x, y: frame.y, width: frame.w, height: frame.h }
    const target = tile.getBoundingClientRect()
    const steps = 24
    const keyframes = Array.from({ length: steps + 1 }, (_, i) => {
      const f = genieFrame(box, target, minimizing ? i / steps : 1 - i / steps)
      return { transformOrigin: '0 0', ...f }
    })
    el.animate(keyframes, { duration, easing: 'linear' })
  }

  // Shared pointer tracking: captures the pointer on el and batches moves into
  // one update per animation frame, so dragging never triggers layout per event.
  function track(
    e: PointerEvent,
    onMove: (ev: PointerEvent) => void,
    onEnd: (ev: PointerEvent) => void,
  ) {
    const el = e.currentTarget as HTMLElement
    el.setPointerCapture(e.pointerId)
    let raf = 0
    let last = e
    const move = (ev: PointerEvent) => {
      last = ev
      if (!raf) {
        raf = requestAnimationFrame(() => {
          raf = 0
          onMove(last)
        })
      }
    }
    const end = (ev: PointerEvent) => {
      cancelAnimationFrame(raf)
      onMove(ev)
      onEnd(ev)
      el.removeEventListener('pointermove', move)
      el.removeEventListener('pointerup', end)
      el.removeEventListener('pointercancel', end)
    }
    el.addEventListener('pointermove', move)
    el.addEventListener('pointerup', end)
    el.addEventListener('pointercancel', end)
  }

  function isControl(target: EventTarget | null) {
    return target instanceof Element && target.closest('button, input, [data-no-drag]') !== null
  }

  function startDrag(e: PointerEvent) {
    if (e.button !== 0 || isControl(e.target)) return
    let origin = { x: e.clientX, y: e.clientY }
    let start: Rect = win.rect
    let dragging = false
    let zone: ReturnType<typeof snapZone> = null

    track(
      e,
      (ev) => {
        const dx = ev.clientX - origin.x
        const dy = ev.clientY - origin.y
        if (!dragging) {
          if (Math.hypot(dx, dy) < DRAG_SLOP) return
          dragging = true
          start = wm.unsnapForDrag(win.id, ev.clientX) ?? win.rect
          origin = { x: ev.clientX, y: ev.clientY }
          return
        }
        wm.setRect(win.id, { ...start, x: start.x + dx, y: start.y + dy })
        zone = snapZone(ev.clientX, ev.clientY, wm.area)
        wm.snapPreview = zone ? zoneRect(zone, wm.area) : null
      },
      () => {
        wm.snapPreview = null
        if (!dragging) return
        if (zone) wm.snap(win.id, zone)
        else wm.commit(win.id)
      },
    )
  }

  function startResize(e: PointerEvent, edge: Edge) {
    if (e.button !== 0) return
    e.stopPropagation()
    wm.focus(win.id)
    wm.unsnapInPlace(win.id)
    const origin = { x: e.clientX, y: e.clientY }
    const start = { ...win.rect }
    track(
      e,
      (ev) =>
        wm.setRect(win.id, resizeRect(start, edge, ev.clientX - origin.x, ev.clientY - origin.y)),
      () => wm.commit(win.id),
    )
  }

  setWindowContext({
    get win() {
      return win
    },
    get focused() {
      return focused
    },
    get theme() {
      return theme
    },
    startDrag,
    toggleMaximize(e: MouseEvent) {
      if (!isControl(e.target)) wm.toggleMaximize(win.id)
    },
  })
</script>

<section
  class="window {theme === 'dark' ? 'glass-dark' : 'glass glass-strong'}"
  class:inactive={!focused}
  class:minimized={win.minimized}
  class:snapped={win.snap !== null}
  class:animating={win.animating}
  aria-label={t.window.label(win.title)}
  aria-hidden={win.minimized}
  style:translate="{frame.x}px {frame.y}px"
  style:width="{frame.w}px"
  style:height="{frame.h}px"
  style:z-index={win.z}
  bind:this={el}
  onpointerdowncapture={() => wm.focus(win.id)}
  in:windowZoom|global={{ appId: win.appId }}
  out:windowZoom|global={{ out: true, appId: win.appId }}
>
  {@render children()}
  {#if win.snap === null}
    {#each edges as edge (edge)}
      <div
        class="handle {edge}"
        aria-hidden="true"
        onpointerdown={(e) => startResize(e, edge)}
      ></div>
    {/each}
  {/if}
</section>

<style>
  .window {
    position: absolute;
    top: 0;
    left: 0;
    display: flex;
    flex-direction: column;
    border-radius: var(--window-radius);
    overflow: hidden;
    transform-origin: center;
    will-change: translate, transform;
    transition:
      opacity var(--dur) var(--ease-out),
      transform var(--dur-slow) var(--ease-out),
      box-shadow var(--dur) var(--ease-out),
      visibility 0s;
  }

  /* Snap and maximize glide to the new frame; drags and resizes stay instant. */
  .window.animating {
    transition:
      translate var(--dur-slow) var(--ease-out),
      width var(--dur-slow) var(--ease-out),
      height var(--dur-slow) var(--ease-out),
      border-radius var(--dur-slow) var(--ease-out),
      opacity var(--dur) var(--ease-out),
      box-shadow var(--dur) var(--ease-out);
  }

  .window.snapped {
    border-radius: 0;
  }

  .glass.inactive {
    box-shadow: var(--glass-shadow-inactive);
  }

  .glass-dark.inactive {
    box-shadow: var(--glass-dark-shadow-inactive);
  }

  /* Minimized: hidden once the genie has poured it into the dock (300ms),
     and no input meanwhile. */
  .window.minimized {
    opacity: 0;
    pointer-events: none;
    visibility: hidden;
    transition: visibility 0s linear 300ms;
  }

  .handle {
    position: absolute;
    z-index: 10;
  }

  .n,
  .s {
    left: 6px;
    right: 6px;
    height: 6px;
    cursor: ns-resize;
  }
  .e,
  .w {
    top: 6px;
    bottom: 6px;
    width: 6px;
    cursor: ew-resize;
  }
  .n {
    top: 0;
  }
  .s {
    bottom: 0;
  }
  .e {
    right: 0;
  }
  .w {
    left: 0;
  }

  .ne,
  .nw,
  .se,
  .sw {
    width: 12px;
    height: 12px;
  }
  .ne {
    top: 0;
    right: 0;
    cursor: nesw-resize;
  }
  .sw {
    bottom: 0;
    left: 0;
    cursor: nesw-resize;
  }
  .nw {
    top: 0;
    left: 0;
    cursor: nwse-resize;
  }
  .se {
    bottom: 0;
    right: 0;
    cursor: nwse-resize;
  }
</style>
