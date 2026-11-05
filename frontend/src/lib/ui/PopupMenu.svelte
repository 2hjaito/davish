<script lang="ts" module>
  export interface Action {
    label: string
    run: () => void
    disabled?: boolean
    danger?: boolean
  }
</script>

<script lang="ts">
  import { portal } from './portal'

  // A menu at a point of the screen, moved to <body> so that nothing clips
  // it: the ⋮ button's menu and every right-click menu.
  interface Props {
    x: number
    y: number
    label: string
    /** null is a divider. */
    actions: readonly (Action | null)[]
    onclose: () => void
    /** The button that opened the menu, which handles its own presses. */
    anchor?: HTMLElement
    /** Hangs to the left of x, as under a button at the right edge. */
    alignRight?: boolean
    /** Opens above y; by default it does when it would leave the window. */
    up?: boolean
  }

  let { x, y, label, actions, onclose, anchor, alignRight = false, up }: Props = $props()

  let menu = $state<HTMLElement>()
  let width = $state(0)
  let height = $state(0)

  const left = $derived(alignRight || x + width > window.innerWidth)
  const above = $derived(up ?? y + height > window.innerHeight)

  $effect(() => {
    requestAnimationFrame(() => menu?.querySelector<HTMLElement>('button:not(:disabled)')?.focus())
  })

  function run(a: Action) {
    a.run()
    onclose()
  }

  function onPointer(e: PointerEvent) {
    if (!menu?.contains(e.target as Node) && !anchor?.contains(e.target as Node)) onclose()
  }
</script>

<svelte:window
  onpointerdown={onPointer}
  onkeydown={(e) => e.key === 'Escape' && onclose()}
  onblur={onclose}
/>

<div
  bind:this={menu}
  bind:clientWidth={width}
  bind:clientHeight={height}
  use:portal
  class="action-menu glass glass-strong"
  role="menu"
  aria-label={label}
  style:left="{x}px"
  style:top="{y}px"
  style:translate="{left ? '-100%' : '0'}
  {above ? '-100%' : '0'}"
>
  {#each actions as a, i (i)}
    {#if a === null}
      <div class="divider"></div>
    {:else}
      <button role="menuitem" class:danger={a.danger} disabled={a.disabled} onclick={() => run(a)}
        >{a.label}</button
      >
    {/if}
  {/each}
</div>

<style>
  .action-menu {
    position: fixed;
    z-index: 3000;
    min-width: 200px;
    padding: 5px;
    border-radius: var(--menu-radius);
    background: var(--glass-bg-menu);
    font-size: var(--text-md);
  }

  .action-menu button {
    width: 100%;
    padding: 5px 10px;
    border: 0;
    border-radius: var(--item-radius);
    background: transparent;
    text-align: left;
  }

  .action-menu button:hover:not(:disabled),
  .action-menu button:focus-visible {
    background: var(--accent);
    color: var(--fg-on-accent);
    outline: none;
  }

  .action-menu button:disabled {
    color: var(--fg-subtle);
  }

  .action-menu button.danger:not(:hover) {
    color: var(--danger);
  }

  .divider {
    height: 1px;
    margin: 4px 8px;
    background: var(--divider);
  }
</style>
