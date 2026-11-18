<script lang="ts">
  import type { Snippet } from 'svelte'

  import { t } from '../i18n'
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
      <svg viewBox="0 0 85.4 85.4" aria-hidden="true">
        <circle class="ring" cx="42.7" cy="42.7" r="42.7" />
        <circle class="fill" cx="42.7" cy="42.7" r="39.1" />
        <path
          class="glyph"
          d="m22.5 57.8 35.3-35.3c1.4-1.4 3.6-1.4 5 0l.1.1c1.4 1.4 1.4 3.6 0 5l-35.3 35.3c-1.4 1.4-3.6 1.4-5 0l-.1-.1c-1.3-1.4-1.3-3.6 0-5zm5.1-35.3 35.3 35.3c1.4 1.4 1.4 3.6 0 5l-.1.1c-1.4 1.4-3.6 1.4-5 0l-35.3-35.3c-1.4-1.4-1.4-3.6 0-5l.1-.1c1.4-1.3 3.6-1.3 5 0z"
        />
      </svg>
    </button>
    <button
      class="light min"
      aria-label={t.window.minimize}
      onclick={() => wm.minimize(ctx.win.id)}
    >
      <svg viewBox="0 0 85.4 85.4" aria-hidden="true">
        <circle class="ring" cx="42.7" cy="42.7" r="42.7" />
        <circle class="fill" cx="42.7" cy="42.7" r="39.1" />
        <path
          class="glyph"
          d="m17.8 39.1h49.9c1.9 0 3.5 1.6 3.5 3.5v.1c0 1.9-1.6 3.5-3.5 3.5h-49.9c-1.9 0-3.5-1.6-3.5-3.5v-.1c0-1.9 1.5-3.5 3.5-3.5z"
        />
      </svg>
    </button>
    <button
      class="light max"
      aria-label={t.window.maximize}
      onclick={() => wm.toggleMaximize(ctx.win.id)}
    >
      <svg viewBox="0 0 85.4 85.4" aria-hidden="true">
        <circle class="ring" cx="42.7" cy="42.7" r="42.7" />
        <circle class="fill" cx="42.7" cy="42.7" r="39.1" />
        <path
          class="glyph"
          d="m31.2 20.8h26.7c3.6 0 6.5 2.9 6.5 6.5v26.7zm23.2 43.7h-26.8c-3.6 0-6.5-2.9-6.5-6.5v-26.8z"
        />
      </svg>
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

  /* Drawn from macOS's own: a darker ring, the fill, and a glyph that
     shows on all three when hovering the group. Only fills change, so a
     hover or press repaints three tiny SVGs and nothing else. */
  .light {
    width: 12px;
    height: 12px;
    border: 0;
    padding: 0;
    background: none;
    display: block;
    border-radius: 50%;
  }

  .light svg {
    display: block;
    width: 100%;
    height: 100%;
  }

  .ring {
    fill: var(--ring);
  }

  .fill {
    fill: var(--fill);
  }

  .glyph {
    fill: var(--glyph);
    opacity: 0;
    transition: opacity var(--dur-fast) var(--ease-out);
  }

  .lights:hover .glyph,
  .light:focus-visible .glyph {
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
    --ring: #e24b41;
    --fill: #ed6a5f;
    --glyph: #460804;
  }
  .close:active {
    --ring: #a14239;
    --fill: #b15048;
    --glyph: #170101;
  }
  .min {
    --ring: #e1a73e;
    --fill: #f6be50;
    --glyph: #90591d;
  }
  .min:active {
    --ring: #a67f36;
    --fill: #b8923b;
    --glyph: #532a0a;
  }
  .max {
    --ring: #2dac2f;
    --fill: #61c555;
    --glyph: #2a6218;
  }
  .max:active {
    --ring: #428234;
    --fill: #4a9741;
    --glyph: #113107;
  }

  /* A window in the background: grey, until the group is hovered. */
  .inactive:not(:hover) .light {
    --ring: var(--tl-off-ring);
    --fill: var(--tl-off-fill);
  }

  /* The sidebar's rail only fits the lights; the Help menu still has this. */
  @container window (max-width: 560px) {
    .titlebar.bare .help {
      display: none;
    }
  }
</style>
