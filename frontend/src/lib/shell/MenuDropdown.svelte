<script lang="ts">
  import Icon from '../ui/Icon.svelte'
  import { comboLabel, type Menu, type MenuItem } from './menus.svelte'
  import Popover from './Popover.svelte'

  interface Props {
    menu: Menu
    open: boolean
    onopen: () => void
    /** Hovering while another menu is open moves to this one, like macOS. */
    onhover: () => void
    onclose: () => void
    /** ←/→ to the neighboring menu. */
    onstep: (dir: -1 | 1) => void
  }

  let { menu, open, onopen, onhover, onclose, onstep }: Props = $props()

  let active = $state(-1)
  const items = $derived(menu.items)

  $effect(() => {
    if (open) active = -1
  })

  function run(it: MenuItem) {
    if (it.disabled) return
    onclose()
    it.run()
  }

  function move(dir: 1 | -1) {
    const n = items.length
    for (let i = 1; i <= n; i++) {
      const j = ((((active < 0 && dir < 0 ? 0 : active) + dir * i) % n) + n) % n
      const it = items[j]
      if (it && !it.disabled) {
        active = j
        return
      }
    }
  }

  function onKeyDown(e: KeyboardEvent) {
    if (!open) return
    const keys: Record<string, () => void> = {
      ArrowDown: () => move(1),
      ArrowUp: () => move(-1),
      ArrowLeft: () => onstep(-1),
      ArrowRight: () => onstep(1),
      Enter: () => {
        const it = items[active]
        if (it) run(it)
      },
    }
    const f = keys[e.key]
    if (!f) return
    e.preventDefault()
    e.stopPropagation()
    f()
  }
</script>

<svelte:window onkeydowncapture={onKeyDown} />

<div class="anchor">
  <button
    class="mbtn"
    class:bold={menu.bold}
    class:open
    aria-haspopup="menu"
    aria-expanded={open}
    onclick={() => (open ? onclose() : onopen())}
    onpointerenter={onhover}
  >
    {menu.label}
  </button>
  <Popover {open} {onclose} label={menu.label} width={260}>
    {#each items as it, i (i)}
      {#if it === null}
        <div class="divider"></div>
      {:else}
        <button
          role="menuitem"
          class="item"
          class:active={i === active}
          disabled={it.disabled}
          onpointerenter={() => (active = it.disabled ? -1 : i)}
          onclick={() => run(it)}
        >
          <span class="check">
            {#if it.checked}<Icon name="check" size={12} strokeWidth={2.6} />{/if}
          </span>
          <span class="text">{it.label}</span>
          {#if it.combo}<span class="key">{comboLabel(it.combo)}</span>{/if}
        </button>
      {/if}
    {/each}
  </Popover>
</div>

<style>
  .anchor {
    position: relative;
  }

  .bold {
    font-weight: 700;
  }

  .mbtn.open {
    background: var(--glass-hover);
  }

  .item {
    width: 100%;
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 5px 10px 5px 4px;
    border: 0;
    border-radius: var(--item-radius);
    background: transparent;
    text-align: left;
    white-space: nowrap;
  }

  .item.active {
    background: var(--accent);
    color: var(--fg-on-accent);
  }

  .item:disabled {
    color: var(--fg-subtle);
  }

  .check {
    width: 16px;
    display: flex;
    justify-content: center;
  }

  .text {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .key {
    font-size: var(--text-xs);
    color: var(--fg-subtle);
    padding-left: 16px;
  }

  .item.active .key {
    color: inherit;
    opacity: 0.85;
  }

  .divider {
    height: 1px;
    background: var(--divider);
    margin: 5px 8px;
  }
</style>
