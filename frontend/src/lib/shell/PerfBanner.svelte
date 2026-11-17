<script lang="ts">
  import { onMount } from 'svelte'

  import { t } from '../i18n'
  import { settings } from '../stores/settings.svelte'
  import { perfCheck } from './perfCheck.svelte'

  onMount(() => {
    if (settings.perfPreset === 'auto') void perfCheck.start(false)
  })

  function apply() {
    settings.usePreset('light')
    perfCheck.hide()
  }

  function later() {
    settings.dismissPerfSuggestion()
    perfCheck.hide()
  }
</script>

{#if perfCheck.suggestion}
  <section class="offer glass glass-strong" role="status">
    <span class="text">
      <strong>{t.settings.perfSuggestTitle}</strong>
      <span>{t.settings.perfSuggestBody}</span>
    </span>
    <span class="actions">
      <button type="button" class="btn" onclick={apply}>{t.settings.perfApply}</button>
      <button type="button" class="later" onclick={later}>{t.settings.perfLater}</button>
    </span>
  </section>
{/if}

<style>
  .offer {
    position: fixed;
    top: calc(var(--menubar-h) + 8px + var(--dock-clear-top, 0px));
    left: 50%;
    translate: -50% 0;
    z-index: 960;
    display: flex;
    align-items: center;
    gap: 16px;
    width: min(440px, calc(100vw - 24px));
    padding: 12px 14px;
    border-radius: var(--menu-radius);
  }

  .text {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
    font-size: var(--text-sm);
    color: var(--fg-muted);
  }

  .text strong {
    color: var(--fg);
  }

  .actions {
    display: flex;
    flex-shrink: 0;
    gap: 6px;
  }

  .later {
    border: 0;
    padding: 4px 10px;
    border-radius: var(--control-radius);
    background: none;
    color: var(--fg-muted);
    font-weight: 600;
  }

  .later:hover {
    background: var(--fill-soft);
    color: var(--fg);
  }
</style>
