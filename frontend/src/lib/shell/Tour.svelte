<script lang="ts">
  import { untrack } from 'svelte'

  import { t } from '../i18n/vi'
  import { load, save } from '../stores/persist'
  import { servers } from '../stores/servers.svelte'
  import { ui } from '../stores/ui.svelte'
  import { portal } from '../ui/portal'

  // What each step of t.tour.steps points at; a step whose target is not
  // on screen (no metrics while offline) is skipped.
  const TARGETS = [
    '.menubar .picker',
    '.menubar .stat',
    '.dock',
    '.menubar .search',
    '.window:not(.minimized)',
    '.menubar .left',
  ]

  interface Box {
    x: number
    y: number
    w: number
    h: number
  }

  let index = $state(0)
  let box = $state<Box | null>(null)

  const steps = $derived(t.tour.steps.map((s, i) => ({ ...s, target: TARGETS[i] ?? '' })))

  // First run: once there is a server to look at, and never again after
  // it is finished or skipped. Help → Hướng dẫn nhanh shows it again.
  $effect(() => {
    if (!servers.ready || servers.list.length === 0) return
    untrack(() => {
      if (!load('tourDone', false)) ui.tour = true
    })
  })

  $effect(() => {
    if (!ui.tour) return
    untrack(() => {
      index = 0
      locate()
    })
  })

  function locate() {
    for (let i = index; i < steps.length; i++) {
      const el = document.querySelector(steps[i]?.target ?? '')
      if (!el) continue
      const r = el.getBoundingClientRect()
      index = i
      box = { x: r.left, y: r.top, w: r.width, h: r.height }
      return
    }
    finish()
  }

  function next() {
    index++
    locate()
  }

  function finish() {
    ui.tour = false
    box = null
    save('tourDone', true)
  }

  function onKeyDown(e: KeyboardEvent) {
    if (!ui.tour) return
    if (e.key === 'Escape') finish()
    else if (e.key === 'Enter' || e.key === 'ArrowRight') next()
    else return
    e.preventDefault()
    e.stopPropagation()
  }

  // The callout goes below a target in the top half, above otherwise.
  const callout = $derived.by(() => {
    if (!box) return ''
    const left = Math.min(Math.max(12, box.x), window.innerWidth - 332)
    return box.y < window.innerHeight / 2
      ? `left:${left}px;top:${box.y + box.h + 12}px`
      : `left:${left}px;bottom:${window.innerHeight - box.y + 12}px`
  })
</script>

<svelte:window onkeydowncapture={onKeyDown} onresize={() => ui.tour && locate()} />

{#if ui.tour && box}
  {@const step = steps[index]}
  <div class="tour" use:portal>
    <div
      class="hole"
      style:left="{box.x - 6}px"
      style:top="{box.y - 4}px"
      style:width="{box.w + 12}px"
      style:height="{box.h + 8}px"
    ></div>
    <div class="callout glass glass-strong" style={callout} role="dialog" aria-label={step?.title}>
      <div class="head">
        <strong>{step?.title}</strong>
        <span class="count">{t.tour.step(index + 1, steps.length)}</span>
      </div>
      <p>{step?.body}</p>
      <div class="actions">
        <button type="button" class="btn" onclick={finish}>{t.tour.skip}</button>
        <!-- svelte-ignore a11y_autofocus -->
        <button type="button" class="btn btn-primary" autofocus onclick={next}>
          {index === steps.length - 1 ? t.tour.done : t.tour.next}
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .tour {
    position: fixed;
    inset: 0;
    z-index: 5000;
  }

  /* The dimmed screen is the hole's shadow, so the target stays lit. */
  .hole {
    position: absolute;
    border-radius: 10px;
    box-shadow: 0 0 0 9999px rgb(0 0 0 / 0.45);
    outline: 2px solid var(--accent);
    transition: all var(--dur) var(--ease-out);
    pointer-events: none;
  }

  .callout {
    position: absolute;
    width: 320px;
    padding: 14px 16px 12px;
    border-radius: 14px;
    font-size: var(--text-sm);
  }

  .head {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
  }

  .count {
    font-size: var(--text-xs);
    color: var(--fg-subtle);
  }

  p {
    margin: 6px 0 12px;
    line-height: 1.4;
  }

  .actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
  }
</style>
