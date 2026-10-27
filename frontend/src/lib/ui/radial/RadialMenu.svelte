<script lang="ts" module>
  import type { IconName } from '../icons'

  export interface RadialItem {
    id: string
    label: string
    icon: IconName
    /** Shown dimmed and never chosen. */
    disabled?: boolean
    /** Choices in the outer ring beyond this petal. */
    children?: { id: string; label: string }[]
  }
</script>

<script lang="ts">
  import { cubicOut } from 'svelte/easing'
  import type { TransitionConfig } from 'svelte/transition'

  import { ms } from '../../motion'
  import { portal } from '../portal'
  import Icon from '../Icon.svelte'
  import { type Hit, petalSpan, polar, RADII, sectorPath } from './radial'

  interface Props {
    /** Center, in viewport pixels: where the drag started. */
    x: number
    y: number
    items: RadialItem[]
    hit: Hit
    /** What is being dragged, shown in the middle. */
    title: string
    icon: IconName
    /** One line under the menu on how to choose or cancel. */
    hint?: string
  }

  let { x, y, items, hit, title, icon, hint = '' }: Props = $props()

  const R = RADII.ring
  const mid = (RADII.inner + RADII.outer) / 2
  const ringMid = (RADII.outer + RADII.ring) / 2
  const open = $derived(hit.petal === null ? undefined : items[hit.petal])
  const subs = $derived(open?.children && !open.disabled ? open.children : [])

  function subSpan(petal: number, i: number, n: number): [number, number] {
    const [a0, a1] = petalSpan(petal, items.length)
    const w = (a1 - a0) / n
    return [a0 + i * w, a0 + (i + 1) * w]
  }

  function at(r: number, deg: number) {
    const p = polar(R, R, r, deg)
    return `left: ${p.x}px; top: ${p.y}px`
  }

  // Zoom and fade in, as DESIGN.md asks of the drag menu.
  function zoom(_node: Element): TransitionConfig {
    return {
      duration: ms(180),
      easing: cubicOut,
      css: (t) => `opacity: ${t}; transform: scale(${0.85 + 0.15 * t})`,
    }
  }
</script>

<div
  class="radial"
  style:left="{x - R}px"
  style:top="{y - R}px"
  style:width="{2 * R}px"
  style:height="{2 * R}px"
  role="menu"
  aria-label={title}
  use:portal
  transition:zoom
>
  <div class="disc glass glass-strong" style:inset="{R - RADII.outer}px"></div>

  <svg width={2 * R} height={2 * R} aria-hidden="true">
    {#each items as item, i (item.id)}
      {@const [a0, a1] = petalSpan(i, items.length)}
      <path
        class="petal"
        class:active={hit.petal === i && hit.sub === null && !item.disabled}
        d={sectorPath(R, R, RADII.inner, RADII.outer, a0 + 1, a1 - 1)}
      />
    {/each}
    {#if hit.petal !== null}
      {#each subs as sub, i (sub.id)}
        {@const [a0, a1] = subSpan(hit.petal, i, subs.length)}
        <path
          class="sub"
          class:active={hit.sub === i}
          d={sectorPath(R, R, RADII.outer + 4, RADII.ring, a0 + 1, a1 - 1)}
        />
      {/each}
    {/if}
    <circle class="hole" cx={R} cy={R} r={RADII.inner - 2} />
  </svg>

  {#each items as item, i (item.id)}
    {@const [a0, a1] = petalSpan(i, items.length)}
    <div
      class="label"
      class:disabled={item.disabled}
      class:active={hit.petal === i && !item.disabled}
      style={at(mid, (a0 + a1) / 2)}
      role="menuitem"
      aria-disabled={item.disabled}
    >
      <Icon name={item.icon} size={17} />
      <span>{item.label}</span>
    </div>
  {/each}

  {#if hit.petal !== null}
    {#each subs as sub, i (sub.id)}
      {@const [a0, a1] = subSpan(hit.petal, i, subs.length)}
      <div
        class="sublabel"
        class:active={hit.sub === i}
        style={at(ringMid, (a0 + a1) / 2)}
        role="menuitem"
      >
        {sub.label}
      </div>
    {/each}
  {/if}

  <div class="center" style={at(0, 0)}><Icon name={icon} size={18} /></div>
  <div class="title glass glass-strong" style:top="{R + RADII.outer + 10}px">
    <strong>{title}</strong>
    {#if hint}<span>{hint}</span>{/if}
  </div>
</div>

<style>
  .radial {
    position: fixed;
    z-index: 960;
    pointer-events: none;
    user-select: none;
  }

  .disc {
    position: absolute;
    border-radius: 50%;
  }

  svg {
    position: absolute;
    inset: 0;
    overflow: visible;
  }

  .petal {
    fill: transparent;
    stroke: var(--divider-soft);
    stroke-width: 1;
    transition: fill var(--dur-fast) var(--ease-out);
  }

  .petal.active,
  .sub.active {
    fill: var(--accent-soft);
    stroke: var(--accent);
  }

  .sub {
    fill: var(--glass-bg-strong);
    stroke: var(--glass-border);
    stroke-width: 1;
    transition: fill var(--dur-fast) var(--ease-out);
  }

  .hole {
    fill: var(--glass-bg-strong);
    stroke: var(--divider-soft);
  }

  .label,
  .sublabel,
  .center {
    position: absolute;
    transform: translate(-50%, -50%);
  }

  .label {
    width: 78px;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 3px;
    text-align: center;
    font-size: var(--text-xs);
    font-weight: 500;
    line-height: 1.2;
    color: var(--fg);
  }

  .label :global(svg) {
    color: var(--accent);
  }

  .label.active {
    color: var(--accent-strong);
    font-weight: 600;
  }

  .label.disabled {
    opacity: 0.4;
  }

  .sublabel {
    font-size: var(--text-xs);
    font-weight: 700;
    letter-spacing: 0.04em;
    color: var(--fg-muted);
  }

  .sublabel.active {
    color: var(--accent-strong);
  }

  .center {
    display: flex;
    color: var(--fg-muted);
  }

  .title {
    position: absolute;
    left: 50%;
    transform: translateX(-50%);
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 2px;
    max-width: 320px;
    padding: 5px 12px;
    border-radius: var(--item-radius);
    font-size: var(--text-sm);
    white-space: nowrap;
  }

  .title strong {
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .title span {
    font-size: var(--text-xs);
    color: var(--fg-muted);
  }
</style>
