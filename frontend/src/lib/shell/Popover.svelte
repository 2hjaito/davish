<script lang="ts">
  import type { Snippet } from 'svelte'

  import { fadeOut, popIn } from '../motion'

  interface Props {
    open: boolean
    onclose: () => void
    label: string
    width?: number
    align?: 'left' | 'right'
    children: Snippet
  }

  let { open, onclose, label, width = 270, align = 'left', children }: Props = $props()

  let el = $state<HTMLElement>()

  function onPointerDown(e: PointerEvent) {
    // The panel shares an anchor with its toggle button, which closes it itself;
    // anything else, including another menu's toggle, closes it here.
    const anchor = el?.parentElement
    if (open && anchor && !anchor.contains(e.target as Node)) onclose()
  }

  function onKeyDown(e: KeyboardEvent) {
    if (open && e.key === 'Escape') onclose()
  }
</script>

<svelte:window onpointerdown={onPointerDown} onkeydown={onKeyDown} />

{#if open}
  <div
    bind:this={el}
    class="popover glass"
    class:right={align === 'right'}
    role="menu"
    aria-label={label}
    style:width="{width}px"
    in:popIn
    out:fadeOut
  >
    {@render children()}
  </div>
{/if}

<style>
  .popover {
    position: absolute;
    top: calc(100% + 4px);
    left: 0;
    z-index: 100;
    padding: 6px;
    border-radius: var(--menu-radius);
    background: var(--glass-bg-menu);
    font-size: var(--text-md);
    color: var(--fg);
    transform-origin: top left;
  }

  .right {
    left: auto;
    right: 0;
    transform-origin: top right;
  }
</style>
