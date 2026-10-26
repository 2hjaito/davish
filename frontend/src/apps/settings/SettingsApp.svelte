<script lang="ts">
  import { t } from '../../lib/i18n/vi'
  import { settings, WALLPAPER_ACCENTS } from '../../lib/stores/settings.svelte'
  import TitleBar from '../../lib/wm/TitleBar.svelte'
  import type { AppProps } from '../types'

  let { win }: AppProps = $props()
</script>

<TitleBar>
  <span class="title">{win.title}</span>
  <span class="spacer"></span>
</TitleBar>

<div class="body">
  <h2>{t.settings.appearance}</h2>

  <label class="row">
    <span>
      <strong>{t.settings.reduceGlass}</strong>
      <small>{t.settings.reduceGlassHint}</small>
    </span>
    <input
      type="checkbox"
      role="switch"
      checked={settings.reduceGlass}
      onchange={(e) => settings.set('reduceGlass', e.currentTarget.checked)}
    />
  </label>

  <label class="row">
    <span>
      <strong>{t.settings.reduceMotion}</strong>
      <small>{t.settings.reduceMotionHint}</small>
    </span>
    <input
      type="checkbox"
      role="switch"
      checked={settings.reduceMotion}
      onchange={(e) => settings.set('reduceMotion', e.currentTarget.checked)}
    />
  </label>

  <div class="row">
    <strong>{t.settings.wallpaper}</strong>
    <div class="swatches" role="radiogroup" aria-label={t.settings.wallpaper}>
      {#each WALLPAPER_ACCENTS as c (c)}
        <button
          class="swatch"
          role="radio"
          aria-checked={settings.wallpaperAccent === c}
          aria-label={t.settings.wallpaperColor(c)}
          style:background={c}
          onclick={() => settings.set('wallpaperAccent', c)}
        ></button>
      {/each}
    </div>
  </div>
</div>

<style>
  .title {
    flex: 1;
    text-align: center;
    font-weight: 600;
  }

  .spacer {
    width: 52px;
  }

  .body {
    flex: 1;
    overflow-y: auto;
    padding: 14px 20px;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  h2 {
    margin: 0 0 6px;
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--fg-muted);
    letter-spacing: 0.04em;
    text-transform: uppercase;
  }

  .row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    padding: 10px 12px;
    border-radius: var(--item-radius);
    background: var(--glass-highlight);
  }

  .row span {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  small {
    color: var(--fg-muted);
    font-size: var(--text-xs);
  }

  /* A macOS-style switch drawn over the native checkbox, which keeps its semantics. */
  input[type='checkbox'] {
    appearance: none;
    flex-shrink: 0;
    position: relative;
    width: 32px;
    height: 18px;
    margin: 0;
    border-radius: 9px;
    background: var(--fill-track);
    transition: background var(--dur) var(--ease-out);
  }

  input[type='checkbox']::after {
    content: '';
    position: absolute;
    top: 2px;
    left: 2px;
    width: 14px;
    height: 14px;
    border-radius: 50%;
    background: var(--switch-knob);
    box-shadow: var(--switch-knob-shadow);
    transition: translate var(--dur) var(--ease-out);
  }

  input[type='checkbox']:checked {
    background: var(--accent);
  }

  input[type='checkbox']:checked::after {
    translate: 14px 0;
  }

  .swatches {
    display: flex;
    gap: 8px;
  }

  .swatch {
    width: 22px;
    height: 22px;
    border-radius: 50%;
    border: 2px solid var(--glass-border);
    padding: 0;
    transition:
      transform var(--dur-fast) var(--ease-spring),
      border-color var(--dur-fast) var(--ease-out);
  }

  .swatch:hover {
    transform: scale(1.12);
  }

  .swatch[aria-checked='true'] {
    border-color: var(--accent);
  }
</style>
