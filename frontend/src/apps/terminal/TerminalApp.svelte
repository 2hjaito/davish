<script lang="ts">
  import type { Terminal } from '@xterm/xterm'

  import { t } from '../../lib/i18n/vi'
  import { servers } from '../../lib/stores/servers.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import { getWindowContext } from '../../lib/wm/context'
  import TitleBar from '../../lib/wm/TitleBar.svelte'
  import { wm } from '../../lib/wm/windowManager.svelte'
  import type { AppProps } from '../types'
  import { MockShell } from './mockShell'
  import XTerm from './XTerm.svelte'

  let { win }: AppProps = $props()

  const ctx = getWindowContext()
  const server = $derived(servers.list.find((s) => s.id === win.serverID))

  interface Tab {
    id: number
    cwd: string
  }

  let nextId = 1
  let tabs = $state<Tab[]>([{ id: nextId++, cwd: '~' }])
  let activeId = $state(1)
  // Component instances stay out of $state; they are only used to focus.
  const views: Record<number, XTerm> = {}

  function title(tab: Tab) {
    return `${server?.user ?? ''}@${server?.name ?? ''}: ${tab.cwd}`
  }

  function connect(tab: Tab, term: Terminal) {
    // TODO(M3): replace MockShell with a PTY session from internal/term.
    const shell = new MockShell({
      user: server?.user ?? 'user',
      host: server?.name ?? 'server',
      banner: t.terminal.mockBanner,
      emit: (data) => term.write(data),
      onCwd: (cwd) => (tab.cwd = cwd),
      onExit: () => closeTab(tab.id),
    })
    term.onData((data) => shell.input(data))
    shell.start()
  }

  function newTab() {
    const tab: Tab = { id: nextId++, cwd: '~' }
    tabs.push(tab)
    activeId = tab.id
  }

  function closeTab(id: number) {
    const i = tabs.findIndex((tb) => tb.id === id)
    if (i === -1) return
    tabs.splice(i, 1)
    const next = tabs[Math.min(i, tabs.length - 1)]
    if (next) activeId = next.id
    else wm.close(win.id)
  }

  // Typing should go to the terminal whenever its window comes to the front.
  $effect(() => {
    if (ctx.focused) views[activeId]?.focus()
  })
</script>

<TitleBar>
  <div class="tabs" role="tablist">
    {#each tabs as tab (tab.id)}
      <div class="tab" class:active={tab.id === activeId} class:closable={tabs.length > 1}>
        <button
          role="tab"
          class="label"
          aria-selected={tab.id === activeId}
          onclick={() => (activeId = tab.id)}
        >
          {title(tab)}
        </button>
        {#if tabs.length > 1}
          <button class="close" aria-label={t.terminal.closeTab} onclick={() => closeTab(tab.id)}>
            <Icon name="close" size={10} strokeWidth={2.5} />
          </button>
        {/if}
      </div>
    {/each}
    <button class="add" aria-label={t.terminal.newTab} onclick={newTab}>
      <Icon name="plus" size={12} strokeWidth={2.2} />
    </button>
  </div>
  <span class="spacer"></span>
</TitleBar>

<div class="screen">
  {#each tabs as tab (tab.id)}
    <XTerm
      bind:this={views[tab.id]}
      active={tab.id === activeId}
      onready={(term) => connect(tab, term)}
    />
  {/each}
</div>

<style>
  .tabs {
    flex: 1;
    min-width: 0;
    display: flex;
    justify-content: center;
    align-items: center;
    gap: 4px;
    font-size: 12px;
  }

  .tab {
    position: relative;
    display: flex;
    align-items: center;
    min-width: 0;
    border-radius: var(--control-radius);
    color: var(--fg-on-dark-muted);
    transition: background var(--dur-fast) var(--ease-out);
  }

  .tab:hover {
    background: var(--tab-hover-dark);
  }

  .tab.active {
    background: var(--tab-active-dark);
    color: var(--fg-on-dark);
  }

  .label {
    border: 0;
    background: none;
    padding: 4px 12px;
    color: inherit;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  /* Room for the close button on both sides keeps the title centered. */
  .closable .label {
    padding-inline: 24px;
  }

  .close {
    position: absolute;
    left: 5px;
    display: flex;
    border: 0;
    padding: 2px;
    border-radius: 4px;
    background: none;
    color: inherit;
    opacity: 0;
    transition: opacity var(--dur-fast) var(--ease-out);
  }

  .tab:hover .close,
  .close:focus-visible {
    opacity: 1;
  }

  .close:hover,
  .add:hover {
    background: var(--tab-active-dark);
  }

  .add {
    display: flex;
    border: 0;
    padding: 4px;
    border-radius: var(--control-radius);
    background: none;
    color: var(--fg-on-dark-muted);
  }

  .spacer {
    width: 52px;
  }

  .screen {
    position: relative;
    flex: 1;
    min-height: 0;
  }
</style>
