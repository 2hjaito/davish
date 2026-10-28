<script lang="ts">
  import type { Snippet } from 'svelte'

  import { t } from '../i18n/vi'
  import { ui } from '../stores/ui.svelte'
  import Icon from '../ui/Icon.svelte'
  import { getWindowContext } from './context'
  import { wm } from './windowManager.svelte'

  interface Props {
    children?: Snippet
    /** No bottom border, for a sidebar that runs the full window height (Files). */
    bare?: boolean
  }

  let { children, bare = false }: Props = $props()

  const ctx = getWindowContext()
</script>

<!-- Pointer handlers here only add dragging; every action has a button or shortcut. -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  class="titlebar"
  class:dark={ctx.theme === 'dark'}
  class:bare
  onpointerdown={ctx.startDrag}
  ondblclick={ctx.toggleMaximize}
>
  <div class="lights" class:inactive={!ctx.focused}>
    <button class="light close" aria-label={t.window.close} onclick={() => wm.close(ctx.win.id)}>
      <Icon name="close" size={12} strokeWidth={2.6} />
    </button>
    <button
      class="light min"
      aria-label={t.window.minimize}
      onclick={() => wm.minimize(ctx.win.id)}
    >
      <Icon name="minimize" size={12} strokeWidth={2.6} />
    </button>
    <button
      class="light max"
      aria-label={t.window.maximize}
      onclick={() => wm.toggleMaximize(ctx.win.id)}
    >
      <Icon name="maximize" size={9} strokeWidth={2.8} />
    </button>
  </div>
  {@render children?.()}
  {#if t.help.apps[ctx.win.appId]}
    <button
      class="help"
      aria-label={t.help.button}
      title={t.help.button}
      onclick={() => (ui.help = { kind: 'app', appId: ctx.win.appId })}
    >
      <Icon name="help" size={14} />
    </button>
  {/if}
</div>

<style>
  .titlebar {
    height: var(--window-titlebar-h);
    flex-shrink: 0;
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 0 14px;
    border-bottom: 1px solid var(--divider-soft);
  }

  .titlebar.dark {
    border-bottom-color: var(--divider-dark);
  }

  .titlebar.bare {
    border-bottom: 0;
  }

  .lights {
    display: flex;
    gap: 8px;
  }

  .light {
    width: 12px;
    height: 12px;
    border-radius: 50%;
    border: 0;
    padding: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--tl-glyph);
    transition:
      background var(--dur) var(--ease-out),
      filter var(--dur-fast) var(--ease-out);
  }

  .light:active {
    filter: brightness(0.8);
  }

  /* Like macOS: glyphs appear on all three when hovering the group. */
  .light :global(svg) {
    opacity: 0;
    transition: opacity var(--dur-fast) var(--ease-out);
  }

  .lights:hover .light :global(svg) {
    opacity: 1;
  }

  .help {
    margin-left: auto;
    flex-shrink: 0;
    display: flex;
    padding: 3px;
    border: 0;
    border-radius: 6px;
    background: none;
    color: var(--fg-subtle);
  }

  .help:hover {
    color: var(--fg);
    background: var(--glass-hover);
  }

  .close {
    background: var(--tl-close);
  }
  .min {
    background: var(--tl-min);
  }
  .max {
    background: var(--tl-max);
  }

  .inactive:not(:hover) .light {
    background: var(--tl-inactive);
  }
</style>
