<script lang="ts" module>
  export interface Action {
    label: string
    run: () => void
    disabled?: boolean
    danger?: boolean
  }
</script>

<script lang="ts">
  import Icon from './Icon.svelte'
  import { portal } from './portal'

  // The ⋮ button of a table row. The menu is moved to <body> and placed at
  // the button, so a scrolling table does not clip it.
  interface Props {
    label: string
    actions: readonly (Action | null)[]
  }

  let { label, actions }: Props = $props()

  let pos = $state<{ x: number; y: number } | null>(null)
  let menu = $state<HTMLElement>()
  let button = $state<HTMLButtonElement>()

  function toggle(e: MouseEvent) {
    e.stopPropagation()
    if (pos) {
      pos = null
      return
    }
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect()
    // Opens below the button, or above when it would leave the window.
    const below = r.bottom + 4
    pos = { x: r.right, y: below + 220 > window.innerHeight ? r.top - 4 : below }
    requestAnimationFrame(() => menu?.querySelector<HTMLElement>('button:not(:disabled)')?.focus())
  }

  function run(a: Action) {
    a.run()
    pos = null
  }

  function onPointer(e: PointerEvent) {
    if (pos && !menu?.contains(e.target as Node) && !button?.contains(e.target as Node)) pos = null
  }
</script>

<svelte:window
  onpointerdown={onPointer}
  onkeydown={(e) => pos && e.key === 'Escape' && (pos = null)}
  onblur={() => (pos = null)}
/>

<button
  bind:this={button}
  class="icon-btn"
  aria-label={label}
  aria-haspopup="menu"
  aria-expanded={pos !== null}
  onclick={toggle}
>
  <Icon name="more" size={15} />
</button>

{#if pos}
  <div
    bind:this={menu}
    use:portal
    class="action-menu glass glass-strong"
    role="menu"
    aria-label={label}
    style:left="{pos.x}px"
    style:top="{pos.y}px"
    class:up={pos.y < (button?.getBoundingClientRect().top ?? 0)}
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
{/if}

<style>
  .action-menu {
    position: fixed;
    z-index: 3000;
    translate: -100% 0;
    min-width: 200px;
    padding: 5px;
    border-radius: var(--menu-radius);
    background: var(--glass-bg-menu);
    font-size: var(--text-md);
  }

  .action-menu.up {
    translate: -100% -100%;
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
