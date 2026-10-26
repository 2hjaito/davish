<script lang="ts">
  import { getApp } from '../../apps/registry'
  import { hasRuntime, quit } from '../api/runtime'
  import { t } from '../i18n/vi'
  import { servers } from '../stores/servers.svelte'
  import { sleep } from '../stores/sleep.svelte'
  import Icon from '../ui/Icon.svelte'
  import { wm } from '../wm/windowManager.svelte'
  import { isMac, shortcut } from './platform'
  import Popover from './Popover.svelte'

  let open = $state(false)

  function run(action: () => void) {
    open = false
    action()
  }

  function openSettings() {
    const app = getApp('settings')
    if (app) wm.openOrFocus(app.id, servers.activeId, { title: app.label, size: app.size })
  }
</script>

<div class="anchor">
  <button
    class="mbtn icon"
    aria-label={t.menubar.system}
    aria-haspopup="menu"
    aria-expanded={open}
    onclick={() => (open = !open)}
  >
    <Icon name="server" size={16} />
  </button>

  <Popover {open} onclose={() => (open = false)} label={t.menubar.system} width={240}>
    <button role="menuitem" class="item" onclick={() => run(openSettings)}>
      <Icon name="settings" size={15} />
      <span class="text">{t.system.settings}</span>
      <span class="key">{shortcut(',')}</span>
    </button>
    <div class="divider"></div>
    <button role="menuitem" class="item" onclick={() => run(() => sleep.start())}>
      <Icon name="moon" size={15} />
      <span class="text">{t.system.sleep}</span>
    </button>
    <div class="divider"></div>
    <button role="menuitem" class="item" disabled={!hasRuntime} onclick={() => run(quit)}>
      <Icon name="power" size={15} />
      <span class="text">{t.system.quit}</span>
      {#if isMac}<span class="key">{shortcut('Q')}</span>{/if}
    </button>
  </Popover>
</div>

<style>
  .anchor {
    position: relative;
  }

  .icon {
    display: flex;
    align-items: center;
    padding: 2px 8px;
  }

  .item {
    width: 100%;
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 6px 10px;
    border: 0;
    border-radius: var(--item-radius);
    background: transparent;
    text-align: left;
    transition: background var(--dur-fast) var(--ease-out);
  }

  .item:hover:not(:disabled) {
    background: var(--accent);
    color: var(--fg-on-accent);
  }

  .item:disabled {
    color: var(--fg-subtle);
  }

  .text {
    flex: 1;
  }

  .key {
    font-size: var(--text-xs);
    color: var(--fg-subtle);
  }

  .item:hover:not(:disabled) .key {
    color: inherit;
    opacity: 0.85;
  }

  .divider {
    height: 1px;
    background: var(--divider);
    margin: 5px 8px;
  }
</style>
