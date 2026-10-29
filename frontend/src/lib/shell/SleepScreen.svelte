<script lang="ts">
  import { locale, t } from '../i18n'
  import { fadeOut } from '../motion'
  import { sleep } from '../stores/sleep.svelte'

  let now = $state(new Date())

  $effect(() => {
    if (!sleep.active) return
    now = new Date()
    const timer = setInterval(() => (now = new Date()), 1000)
    return () => clearInterval(timer)
  })

  function wake(e: Event) {
    if (!sleep.active) return
    e.preventDefault()
    e.stopPropagation()
    sleep.wake()
  }
</script>

<!-- Capture phase, so the wake-up key or click never reaches the app below. -->
<svelte:window onkeydowncapture={wake} />

{#if sleep.active}
  <!-- Any key also wakes it (window handler above), so the click is not the only way. -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="sleep" onpointerdown={wake} transition:fadeOut>
    <div class="clock">
      {now.toLocaleTimeString(locale, { hour: '2-digit', minute: '2-digit' })}
    </div>
    <div class="date">
      {now.toLocaleDateString(locale, { weekday: 'long', day: 'numeric', month: 'long' })}
    </div>
    <div class="hint">{t.system.wakeHint}</div>
  </div>
{/if}

<style>
  .sleep {
    position: fixed;
    inset: 0;
    z-index: 5000;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 6px;
    background: var(--sleep-bg);
    color: var(--fg-on-dark);
    -webkit-backdrop-filter: var(--glass-dark-blur);
    backdrop-filter: var(--glass-dark-blur);
    /* The OS may still drag the window by the area under the menu bar. */
    --wails-draggable: no-drag;
  }

  .clock {
    font-size: 72px;
    font-weight: 300;
    font-variant-numeric: tabular-nums;
    letter-spacing: -0.02em;
  }

  .date {
    font-size: var(--text-lg);
    color: var(--fg-on-dark-muted);
  }

  .hint {
    position: absolute;
    bottom: 48px;
    font-size: var(--text-sm);
    color: var(--fg-on-dark-muted);
  }
</style>
