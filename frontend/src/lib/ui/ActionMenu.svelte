<script lang="ts" module>
  import type { Action as MenuAction } from './PopupMenu.svelte'

  export type Action = MenuAction
</script>

<script lang="ts">
  import Icon from './Icon.svelte'
  import PopupMenu from './PopupMenu.svelte'

  // The ⋮ button of a table row.
  interface Props {
    label: string
    actions: readonly (Action | null)[]
  }

  let { label, actions }: Props = $props()

  let pos = $state<{ x: number; y: number; up: boolean } | null>(null)
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
    const up = below + 220 > window.innerHeight
    pos = { x: r.right, y: up ? r.top - 4 : below, up }
  }
</script>

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
  <PopupMenu
    x={pos.x}
    y={pos.y}
    up={pos.up}
    alignRight
    {label}
    {actions}
    anchor={button}
    onclose={() => (pos = null)}
  />
{/if}
