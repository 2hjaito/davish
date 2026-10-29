<script lang="ts">
  import type { Terminal } from '@xterm/xterm'
  import { untrack } from 'svelte'

  import { onTermData, onTermExit } from '../../lib/api/events'
  import { hasRuntime } from '../../lib/api/runtime'
  import { decodeOutput, termClose, termResize, termStart, termWrite } from '../../lib/api/terminal'
  import { t } from '../../lib/i18n'
  import { servers } from '../../lib/stores/servers.svelte'
  import { settings } from '../../lib/stores/settings.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import { getWindowContext } from '../../lib/wm/context'
  import { type TerminalRequest, terminalRequests } from '../../lib/wm/intents.svelte'
  import TitleBar from '../../lib/wm/TitleBar.svelte'
  import { wm } from '../../lib/wm/windowManager.svelte'
  import type { AppProps } from '../types'
  import { MockShell } from './mockShell'
  import XTerm from './XTerm.svelte'

  let { win }: AppProps = $props()

  const ctx = getWindowContext()
  const server = $derived(servers.list.find((s) => s.id === win.serverID))
  const online = $derived(server?.state === 'connected')

  interface Tab {
    id: number
    cwd: string
    /** Where the shell starts, when another app asked for a folder. */
    startDir?: string
    /** Typed at the first prompt, not run. */
    startInput?: string
    /** Title the remote shell sets (bash sets "user@host: cwd" for xterm). */
    title?: string
    /** Remote PTY session, while one is open. */
    session?: string
    starting?: boolean
    /** The session dropped or failed to open; waits for the user to reopen it. */
    lost?: 'dropped' | 'failed'
    /** Set once its xterm exists; reactive, unlike the terms map below. */
    ready?: boolean
  }

  let nextId = 1
  const tabAt = (req?: TerminalRequest): Tab => ({
    id: nextId++,
    cwd: req?.dir ?? '~',
    startDir: req?.dir,
    startInput: req?.input,
  })
  // A window opened to show a folder starts there instead of at home.
  const asked = untrack(() => terminalRequests.take(win.serverID))
  let tabs = $state<Tab[]>(asked.length > 0 ? asked.map(tabAt) : [tabAt()])
  let activeId = $state(1)
  // Component and xterm instances stay out of $state, which would proxy them.
  const views: Record<number, XTerm> = {}
  // eslint-disable-next-line svelte/prefer-svelte-reactivity -- holds xterm instances; Tab.ready carries the reactivity
  const terms = new Map<number, Terminal>()

  // Undo modes a dead full-screen program left on (alternate screen, mouse
  // reporting, hidden cursor) so the notice and the next shell show normally.
  const RESTORE = '\x1b[?1049l\x1b[?1000l\x1b[?1002l\x1b[?1003l\x1b[?1006l\x1b[!p'

  function title(tab: Tab) {
    return tab.title || `${server?.user ?? ''}@${server?.name ?? ''}: ${tab.cwd}`
  }

  function connect(tab: Tab, term: Terminal) {
    if (!hasRuntime) {
      mock(tab, term)
      return
    }
    terms.set(tab.id, term)
    term.onData((data) => {
      if (tab.session) void termWrite(tab.session, data)
    })
    term.onResize(({ cols, rows }) => {
      if (tab.session) void termResize(tab.session, cols, rows)
    })
    term.onTitleChange((title) => (tab.title = title))
    tab.ready = true
    if (!online) term.write(`\x1b[2m${t.terminal.waiting}\x1b[0m\r\n`)
  }

  // The browser preview has no Go side: run the scripted shell instead.
  function mock(tab: Tab, term: Terminal) {
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

  // Open a remote shell for every new or reopened tab while the server is up.
  // A lost session stays closed until the user reopens it, like a terminal app
  // whose process ended: its output stays readable.
  $effect(() => {
    if (!hasRuntime || !online) return
    for (const tab of tabs) {
      const term = terms.get(tab.id)
      if (!tab.ready || !term || tab.session || tab.starting || tab.lost) continue
      tab.starting = true
      termStart(win.serverID, term.cols, term.rows, settings.terminalPrompt, tab.startDir)
        .then((id) => {
          tab.session = id
          // Typed once the prompt is drawn: input written earlier is echoed
          // by the terminal before the prompt, so it looks misplaced.
          if (tab.startInput) typeWhenQuiet(tab, 3000)
        })
        .catch(() => {
          term.write(`\r\n\x1b[31m${t.terminal.startFailed}\x1b[0m\r\n`)
          tab.lost = 'failed'
        })
        .finally(() => (tab.starting = false))
    }
  })

  function reopen(tab: Tab) {
    tab.lost = undefined
    const term = terms.get(tab.id)
    term?.write('\r\n')
    if (!online) {
      term?.write(`\x1b[2m${t.terminal.waiting}\x1b[0m\r\n`)
      void servers.ensureConnected(win.serverID)
    }
    term?.focus()
  }

  $effect(() => {
    if (!hasRuntime) return
    const offData = onTermData(({ id, b64 }) => {
      const tab = tabs.find((x) => x.session === id)
      if (!tab) return
      terms.get(tab.id)?.write(decodeOutput(b64))
      if (tab.startInput) typeWhenQuiet(tab, 250)
    })
    const offExit = onTermExit(({ id, code }) => {
      const tab = tabs.find((x) => x.session === id)
      if (!tab) return
      tab.session = undefined
      // -1: the connection dropped, not the shell; keep the tab to reopen it.
      if (code === -1) {
        terms.get(tab.id)?.write(`${RESTORE}\r\n\x1b[33m${t.terminal.lost}\x1b[0m\r\n`)
        tab.lost = 'dropped'
      } else closeTab(tab.id)
    })
    return () => {
      offData()
      offExit()
      for (const tab of tabs) if (tab.session) void termClose(tab.session)
    }
  })

  // Pending start input per tab: typed after ms without output, which is
  // when the shell has finished drawing its prompt.
  // eslint-disable-next-line svelte/prefer-svelte-reactivity -- bookkeeping, never rendered
  const quiet = new Map<number, ReturnType<typeof setTimeout>>()
  function typeWhenQuiet(tab: Tab, ms: number) {
    clearTimeout(quiet.get(tab.id))
    quiet.set(
      tab.id,
      setTimeout(() => {
        quiet.delete(tab.id)
        if (tab.session && tab.startInput) void termWrite(tab.session, tab.startInput)
        tab.startInput = undefined
      }, ms),
    )
  }

  function newTab(req?: TerminalRequest) {
    const tab = tabAt(req)
    tabs.push(tab)
    activeId = tab.id
  }

  // "Open Terminal here" from Files, while this window is already open.
  $effect(() => {
    if (!terminalRequests.list.some((r) => r.serverID === win.serverID)) return
    untrack(() => {
      for (const req of terminalRequests.take(win.serverID)) newTab(req)
    })
  })

  function closeTab(id: number) {
    const i = tabs.findIndex((tb) => tb.id === id)
    if (i === -1) return
    const session = tabs[i]?.session
    if (session) void termClose(session)
    terms.delete(id)
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
    <button class="add" aria-label={t.terminal.newTab} onclick={() => newTab()}>
      <Icon name="plus" size={12} strokeWidth={2.2} />
    </button>
  </div>
  <span class="spacer"></span>
</TitleBar>

<div class="screen" class:solid={settings.terminalGpu}>
  {#each tabs as tab (tab.id)}
    <XTerm
      bind:this={views[tab.id]}
      active={tab.id === activeId}
      onready={(term) => connect(tab, term)}
    />
    {#if tab.lost && tab.id === activeId}
      <div class="lost glass-dark" role="status">
        <Icon name="warning" size={13} />
        <span>{tab.lost === 'failed' ? t.terminal.startFailed : t.terminal.lostBar}</span>
        <button onclick={() => reopen(tab)}>{t.terminal.reopen}</button>
      </div>
    {/if}
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

  /* GPU mode draws on a solid background; fill the margins around it too. */
  .solid {
    background: var(--term-bg-solid);
  }

  .lost {
    position: absolute;
    left: 50%;
    bottom: 14px;
    translate: -50% 0;
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 6px 6px 12px;
    border-radius: var(--control-radius);
    font-size: 12px;
    white-space: nowrap;
  }

  .lost :global(svg) {
    color: var(--term-warn);
  }

  .lost button {
    border: 0;
    padding: 4px 12px;
    border-radius: var(--control-radius);
    background: var(--accent);
    color: var(--fg-on-accent);
    font-weight: 500;
  }

  .lost button:hover {
    filter: brightness(1.08);
  }
</style>
