<script lang="ts">
  import { isWindowMaximized, minimizeWindow, quit, toggleMaximizeWindow } from '../api/runtime'
  import { t } from '../i18n'
  import Icon from '../ui/Icon.svelte'

  // Windows and Linux run Davish frameless, so the OS draws no buttons to
  // minimize, restore or close it; these stand in, at the menu bar's end.
  let maximized = $state(true)

  async function sync() {
    maximized = await isWindowMaximized()
  }

  $effect(() => {
    void sync()
    window.addEventListener('resize', sync)
    return () => window.removeEventListener('resize', sync)
  })
</script>

<div class="controls" role="group" aria-label={t.menubar.windowControls}>
  <button aria-label={t.menubar.minimize} title={t.menubar.minimize} onclick={minimizeWindow}>
    <Icon name="minimize" size={14} strokeWidth={1.8} />
  </button>
  <button
    aria-label={maximized ? t.menubar.restore : t.menubar.maximize}
    title={maximized ? t.menubar.restore : t.menubar.maximize}
    onclick={toggleMaximizeWindow}
  >
    <Icon name={maximized ? 'copy' : 'stop'} size={12} strokeWidth={1.8} />
  </button>
  <button class="close" aria-label={t.menubar.close} title={t.menubar.close} onclick={quit}>
    <Icon name="close" size={15} strokeWidth={1.8} />
  </button>
</div>

<style>
  /* Flush with the bar's right edge, like the caption buttons of Windows. */
  .controls {
    display: flex;
    align-self: stretch;
    margin-right: -12px;
  }

  button {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 44px;
    padding: 0;
    border: 0;
    background: transparent;
    color: var(--fg);
    transition: background var(--dur-fast) var(--ease-out);
  }

  button:hover {
    background: var(--fill-soft);
  }

  .close:hover {
    background: var(--danger);
    color: #fff;
  }
</style>
