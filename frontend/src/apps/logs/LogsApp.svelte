<script lang="ts">
  import { untrack } from 'svelte'

  import { toUserError } from '../../lib/api/errors'
  import type { LogKind, LogSource } from '../../lib/api/logs'
  import { t } from '../../lib/i18n'
  import { load, save } from '../../lib/stores/persist'
  import { servers } from '../../lib/stores/servers.svelte'
  import Empty from '../../lib/ui/Empty.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import NeedsRoot from '../../lib/ui/NeedsRoot.svelte'
  import { getWindowContext } from '../../lib/wm/context'
  import { logsRequests, openFileInEditor } from '../../lib/wm/intents.svelte'
  import TitleBar from '../../lib/wm/TitleBar.svelte'
  import type { AppProps } from '../types'
  import { backend as dockerBackend } from '../docker/backend'
  import { backend as servicesBackend } from '../services/backend'
  import { append, COMMON_FILES, filterLines, normalize, targetProblem } from './buffer'
  import LogView from './LogView.svelte'
  import { follow } from './stream'

  let { win }: AppProps = $props()

  const ctx = getWindowContext()
  const KINDS: LogKind[] = ['journal', 'file', 'docker']
  /** How long a journal may stay empty before the reasons are shown. */
  const QUIET_MS = 5000

  // The source being typed, remembered per server.
  const saved = load(`logs:${untrack(() => win.serverID)}`, {
    kind: 'journal' as LogKind,
    target: '',
  })
  let kind = $state<LogKind>(KINDS.includes(saved.kind) ? saved.kind : 'journal')
  let target = $state(saved.target)
  /** What is being followed, or was last. */
  let source = $state.raw<LogSource | null>(null)
  let lines = $state.raw<string[]>([])
  /** Lines that came in while paused; shown on resume. */
  let held = $state.raw<string[]>([])
  let paused = $state(false)
  let query = $state('')
  let status = $state<'idle' | 'live' | 'stopped' | 'lost' | 'failed'>('idle')
  let failure = $state('')
  let sudo = $state('')
  let suggestions = $state.raw<string[]>([])
  /** The sudo command to offer once a journal stays silent. */
  let quiet = $state('')
  let stop: (() => void) | null = null
  let quietTimer: ReturnType<typeof setTimeout> | undefined

  const server = $derived(servers.list.find((s) => s.id === win.serverID))
  const connected = $derived(server?.state === 'connected')
  const shown = $derived(filterLines(lines, query))
  const problem = $derived(targetProblem(kind, target))
  const listId = $props.id()

  $effect(() => save(`logs:${win.serverID}`, { kind, target }))

  function label(src: LogSource): string {
    return src.kind === 'journal' && !src.target ? t.logs.wholeJournal : src.target
  }

  async function start(src: LogSource) {
    stop?.()
    stop = null
    source = src
    lines = []
    held = []
    paused = false
    failure = ''
    sudo = ''
    quiet = ''
    clearTimeout(quietTimer)
    status = 'live'
    const mine = src
    try {
      const s = await follow(win.serverID, src, {
        lines: (batch) => {
          if (source !== mine) return
          quiet = ''
          if (paused) held = append(held, batch)
          else lines = append(lines, batch)
        },
        end: (e) => {
          if (source !== mine) return
          stop = null
          if (!e.error) status = 'stopped'
          else if (e.error.code === 'CONN_NOT_CONNECTED') status = 'lost'
          else {
            status = 'failed'
            failure = toUserError(e).message
            sudo = e.error.code === 'CMD_NEEDS_ROOT' ? (e.sudo ?? '') : ''
          }
        },
      })
      if (source === mine && status === 'live') {
        stop = s.stop
        // journalctl prints nothing, not an error, for an account that
        // cannot read the system journal; say so instead of waiting on.
        if (src.kind === 'journal')
          quietTimer = setTimeout(() => {
            if (source === mine && status === 'live' && lines.length === 0 && held.length === 0)
              quiet = s.sudo
          }, QUIET_MS)
      } else s.stop()
    } catch (e) {
      if (source !== mine) return
      const ue = toUserError(e)
      status = ue.code === 'CONN_NOT_CONNECTED' ? 'lost' : 'failed'
      failure = ue.message
    }
  }

  function submit(e: SubmitEvent) {
    e.preventDefault()
    if (problem || !connected) return
    void start(normalize({ kind, target }))
  }

  function halt() {
    stop?.()
    stop = null
    status = 'stopped'
  }

  function togglePause() {
    if (paused) {
      lines = append(lines, held)
      held = []
    }
    paused = !paused
  }

  // Requests from Services and Docker, now and while the window is open.
  $effect(() => {
    const req = logsRequests.list.find((r) => r.serverID === win.serverID)
    if (!req) return
    untrack(() => {
      const [last] = logsRequests.take(win.serverID).slice(-1)
      if (!last) return
      kind = last.source.kind
      target = last.source.target
      if (connected) void start(last.source)
      else source = last.source
    })
  })

  // A stream cut by a lost connection carries on once the server is back,
  // as ARCHITECTURE.md §3.1 asks of log tails.
  $effect(() => {
    if (!connected) return
    untrack(() => {
      if (source && (status === 'lost' || (status === 'idle' && source))) void start(source)
    })
  })

  $effect(() => () => {
    stop?.()
    clearTimeout(quietTimer)
  })

  async function suggest() {
    try {
      if (kind === 'journal')
        suggestions = (await servicesBackend.list(win.serverID)).map((u) => u.name)
      else if (kind === 'docker')
        suggestions = (await dockerBackend.list(win.serverID)).map((c) => c.name)
      else suggestions = [...COMMON_FILES]
    } catch {
      suggestions = kind === 'file' ? [...COMMON_FILES] : []
    }
  }
</script>

<TitleBar>
  <span class="title">{source ? label(source) : t.apps.logs}</span>
</TitleBar>

<div class="logs" class:blurred={!ctx.focused}>
  <form class="toolbar" onsubmit={submit}>
    <div class="segmented" role="group" aria-label={t.logs.kindLabel}>
      {#each KINDS as k (k)}
        <button
          type="button"
          aria-pressed={kind === k}
          onclick={() => {
            kind = k
            suggestions = []
          }}>{t.logs.kinds[k]}</button
        >
      {/each}
    </div>
    <input
      class="target"
      class:bad={target !== '' && problem !== null}
      type="text"
      list="{listId}-targets"
      placeholder={t.logs.targetHint[kind]}
      aria-label={t.logs.targetLabel}
      title={problem && target ? t.logs.problems[problem] : ''}
      spellcheck="false"
      autocomplete="off"
      bind:value={target}
      onfocus={() => void suggest()}
    />
    <datalist id="{listId}-targets">
      {#each suggestions as s (s)}<option value={s}></option>{/each}
    </datalist>
    <button type="submit" class="go" disabled={problem !== null || !connected}>
      {t.logs.follow}
    </button>
    {#if status === 'live'}
      <button
        type="button"
        class="tool"
        aria-label={t.logs.stop}
        title={t.logs.stop}
        onclick={halt}
      >
        <Icon name="close" size={13} strokeWidth={2.4} />
      </button>
    {/if}
    <span class="spacer"></span>
    <label class="filter">
      <Icon name="search" size={12} strokeWidth={2.2} />
      <input
        type="text"
        placeholder={t.logs.filter}
        aria-label={t.logs.filter}
        bind:value={query}
        onkeydown={(e) => e.key === 'Escape' && (query = '')}
      />
    </label>
    <button
      type="button"
      class="tool"
      aria-pressed={paused}
      aria-label={paused ? t.logs.resume : t.logs.pause}
      title={paused ? t.logs.resume : t.logs.pause}
      disabled={!source}
      onclick={togglePause}
    >
      <Icon name={paused ? 'play' : 'pause'} size={13} strokeWidth={2.4} />
    </button>
    <button
      type="button"
      class="tool"
      aria-label={t.logs.clear}
      title={t.logs.clear}
      disabled={lines.length === 0}
      onclick={() => {
        lines = []
        held = []
      }}
    >
      <Icon name="trash" size={13} strokeWidth={2.2} />
    </button>
  </form>

  {#if !source}
    <Empty icon="logs" title={t.logs.empty} hint={t.logs.emptyHint} />
  {:else if status === 'failed' && sudo}
    <div class="root">
      <NeedsRoot serverID={win.serverID} command={sudo} message={t.logs.needsRoot} />
    </div>
  {:else if quiet && status === 'live' && connected && lines.length === 0}
    <div class="root">
      <NeedsRoot serverID={win.serverID} command={quiet} message={t.logs.quiet} />
    </div>
  {:else if lines.length === 0}
    <Empty
      icon="logs"
      title={!connected ? t.logs.offline : status === 'failed' ? failure : t.logs.waiting}
      hint={!connected ? t.logs.offlineHint : ''}
    />
  {:else if shown.length === 0}
    <Empty icon="search" title={t.logs.noMatch} />
  {:else}
    <LogView
      lines={shown}
      follow={!paused}
      onopen={(path, line) => openFileInEditor(win.serverID, path, line)}
    />
  {/if}

  <footer class="status">
    <span class="state {status}" class:paused>
      {#if paused}{t.logs.status.paused(held.length)}
      {:else if status === 'live'}{t.logs.status.live}
      {:else if status === 'lost'}{t.logs.status.lost}
      {:else if status === 'failed'}{t.logs.status.failed(failure)}
      {:else if status === 'stopped'}{t.logs.status.stopped}{/if}
    </span>
    <span class="spacer"></span>
    {#if source}<span>{t.logs.lines(shown.length, lines.length)}</span>{/if}
  </footer>
</div>

<style>
  .title {
    flex: 1;
    min-width: 0;
    text-align: center;
    margin-right: 52px;
    font-size: var(--text-sm);
    font-weight: 600;
    color: var(--fg-on-dark-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .logs {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    color: var(--fg-on-dark);
  }

  /* Inputs shrink first; a window too narrow for that wraps a row
     rather than clipping the tools. */
  .toolbar {
    flex-shrink: 0;
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 6px;
    padding: 6px 10px;
    border-bottom: 1px solid var(--divider-dark);
  }

  .segmented {
    display: flex;
    padding: 2px;
    border-radius: 8px;
    background: var(--tab-hover-dark);
  }

  .segmented button {
    padding: 2px 9px;
    white-space: nowrap;
    border: 0;
    border-radius: 6px;
    background: none;
    font-size: var(--text-xs);
    color: var(--fg-on-dark-muted);
  }

  .segmented button[aria-pressed='true'] {
    background: var(--tab-active-dark);
    color: var(--fg-on-dark);
    font-weight: 600;
  }

  input {
    border: 0;
    outline: none;
    background: transparent;
    color: var(--fg-on-dark);
    font-size: var(--text-sm);
  }

  input::placeholder {
    color: var(--fg-on-dark-muted);
  }

  .target {
    flex: 0 1 220px;
    min-width: 90px;
    padding: 4px 8px;
    border-radius: var(--control-radius);
    background: var(--tab-hover-dark);
    font-family: var(--font-mono);
    font-size: var(--text-xs);
  }

  .target.bad {
    box-shadow: inset 0 0 0 1px var(--danger-on-dark);
  }

  .go {
    padding: 3px 10px;
    border: 0;
    border-radius: var(--control-radius);
    background: var(--accent);
    color: var(--fg-on-accent);
    font-size: var(--text-xs);
    font-weight: 600;
    white-space: nowrap;
  }

  .go:disabled {
    opacity: 0.45;
  }

  .spacer {
    flex: 1;
  }

  .filter {
    flex: 0 1 auto;
    min-width: 0;
    display: flex;
    align-items: center;
    gap: 5px;
    padding: 3px 8px;
    border-radius: 8px;
    background: var(--tab-hover-dark);
    color: var(--fg-on-dark-muted);
  }

  .filter input {
    width: 120px;
    min-width: 50px;
    flex: 1;
  }

  .tool {
    display: flex;
    padding: 5px;
    border: 0;
    border-radius: 50%;
    background: none;
    color: var(--fg-on-dark-muted);
  }

  .tool:hover:not(:disabled) {
    background: var(--tab-hover-dark);
    color: var(--fg-on-dark);
  }

  .tool[aria-pressed='true'] {
    background: var(--accent);
    color: var(--fg-on-accent);
  }

  .tool:disabled {
    opacity: 0.4;
  }

  .logs :global(.empty) {
    color: var(--fg-on-dark-muted);
  }

  .logs :global(.empty strong) {
    color: var(--fg-on-dark);
  }

  /* Fills the body like the log view; the notice keeps its light-theme
     text, so it sits on a light card in this dark window. */
  .root {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    padding: 16px;
    color: var(--fg);
  }

  .root :global([role='alert']) {
    background: var(--content-bg);
  }

  .status {
    flex-shrink: 0;
    display: flex;
    gap: 10px;
    padding: 4px 14px;
    border-top: 1px solid var(--divider-dark);
    font-size: var(--text-xs);
    color: var(--fg-on-dark-muted);
    font-variant-numeric: tabular-nums;
  }

  .state.live {
    color: var(--ok-on-dark);
  }

  .state.lost,
  .state.paused {
    color: var(--warn-on-dark);
  }

  .state.failed {
    color: var(--danger-on-dark);
  }
</style>
