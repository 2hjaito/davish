<script lang="ts">
  // One command on several servers, each in its own terminal tab
  // (docs/adr/0021). A tab that waits for input comes forward so the user
  // answers there; Davish never types a sudo password for a run.
  import type { Terminal } from '@xterm/xterm'
  import { onDestroy, tick, untrack } from 'svelte'
  import { SvelteSet } from 'svelte/reactivity'

  import { toUserError } from '../../lib/api/errors'
  import { onFleetJob, onTermData } from '../../lib/api/events'
  import {
    type FleetCheck,
    type FleetItem,
    type JobState,
    fleetCheck,
    fleetPlanUpdate,
    fleetStart,
    fleetStop,
  } from '../../lib/api/fleet'
  import { hasRuntime } from '../../lib/api/runtime'
  import { decodeOutput, termResize, termWrite } from '../../lib/api/terminal'
  import { t } from '../../lib/i18n'
  import { servers } from '../../lib/stores/servers.svelte'
  import { settings } from '../../lib/stores/settings.svelte'
  import { snippets } from '../../lib/stores/snippets.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import { openTerminalAt } from '../../lib/wm/intents.svelte'
  import XTerm from '../terminal/XTerm.svelte'
  import RunConfirm from './RunConfirm.svelte'
  import { finished, keepTail, mockSteps, nextActive, sudoRefused, tally, waiting } from './run'

  let { onback }: { onback: () => void } = $props()

  const r = t.servers.run

  // Choosing.
  const picked = new SvelteSet<string>()
  let command = $state('')
  let notice = $state('')
  let busy = $state(false)
  const online = $derived(servers.list.filter((s) => s.state === 'connected'))
  const nameOf = (id: string) => servers.list.find((s) => s.id === id)?.name ?? id

  // Waiting for the user to confirm.
  let pending = $state<{ items: FleetItem[]; check: FleetCheck } | null>(null)

  // Running.
  interface Tab {
    serverID: string
    command: string
    state: JobState
    exit: number
    termID?: string
    error?: string
    /** The end of the output, to spot a prompt. */
    tail: string
    lastAt: number
    /** sudo -n said a password is required. */
    refused: boolean
  }
  let tabs = $state<Tab[]>([])
  let active = $state('')
  let runID = ''
  let now = $state(Date.now())
  let lastTyped = 0
  // eslint-disable-next-line svelte/prefer-svelte-reactivity -- xterm instances, never rendered
  const terms = new Map<string, Terminal>()
  // eslint-disable-next-line svelte/prefer-svelte-reactivity -- output before the tab's terminal is ready
  const early = new Map<string, Uint8Array[]>()
  const decoders = new Map<string, TextDecoder>() // eslint-disable-line svelte/prefer-svelte-reactivity

  const running = $derived(tabs.length > 0 && !finished(tabs))
  const counts = $derived(tally(tabs))
  const summary = $derived(
    [
      counts.running + counts.queued > 0 ? r.progress(counts.running, counts.queued) : '',
      r.result(counts.done, counts.failed + counts.stopped),
    ]
      .filter(Boolean)
      .join(' · '),
  )
  const isWaiting = (tab: Tab) => waiting(tab.state, tab.tail, now - tab.lastAt)
  const retry = $derived(tabs.filter((x) => x.state === 'failed' || x.state === 'stopped'))
  const shown = $derived(tabs.find((x) => x.serverID === active))

  // Prepare: check how to confirm, then ask.
  async function ask(items: FleetItem[]) {
    if (items.length === 0) return
    const check = hasRuntime
      ? await fleetCheck(items[0]?.command ?? '', items.length)
      : { destructive: false, typed: items.length >= 3 }
    pending = { items, check }
  }

  function runCommand() {
    const c = command.trim()
    if (!c) return
    notice = ''
    void ask([...picked].map((serverID) => ({ serverID, command: c })))
  }

  async function runUpdate() {
    notice = ''
    busy = true
    try {
      const ids = [...picked]
      const u = hasRuntime
        ? await fleetPlanUpdate(ids)
        : {
            items: ids.map((serverID) => ({
              serverID,
              command: 'sudo -n apt-get update && sudo -n apt-get upgrade',
            })),
            skipped: [],
          }
      if (u.skipped.length > 0)
        notice = r.skipped(
          u.skipped
            .map(
              (s) =>
                `${nameOf(s.serverID)} (${s.reason === 'noManager' ? r.noManager : (t.errors[s.reason as keyof typeof t.errors] ?? s.reason)})`,
            )
            .join(', '),
        )
      await ask(u.items)
    } catch (e) {
      notice = toUserError(e).message
    } finally {
      busy = false
    }
  }

  async function start() {
    const p = pending
    pending = null
    if (!p) return
    for (const term of terms.values()) term.dispose()
    terms.clear()
    early.clear()
    decoders.clear()
    tabs = p.items.map((it) => ({
      serverID: it.serverID,
      command: it.command,
      state: 'queued',
      exit: -1,
      tail: '',
      lastAt: Date.now(),
      refused: false,
    }))
    active = p.items[0]?.serverID ?? ''
    await tick()
    const first = terms.get(active)
    const cols = first?.cols ?? 120
    const rows = first?.rows ?? 30
    if (!hasRuntime) return mock(p.items)
    try {
      const run = await fleetStart(p.items, cols, rows)
      runID = run.id
    } catch (e) {
      notice = toUserError(e).message
      tabs = []
    }
  }

  function stopAll() {
    if (runID && hasRuntime) void fleetStop(runID)
  }

  function again() {
    void ask(retry.map((x) => ({ serverID: x.serverID, command: x.command })))
  }

  function compose() {
    if (running) return
    tabs = []
    runID = ''
  }

  // Events from Go: job states, and output of the run's sessions.
  $effect(() => {
    if (!hasRuntime) return
    const offJob = onFleetJob((j) => {
      if (j.runID !== runID) return
      const tab = tabs.find((x) => x.serverID === j.serverID)
      if (!tab) return
      tab.state = j.state
      tab.exit = j.exit
      tab.error = j.error
      if (j.termID && !tab.termID) {
        tab.termID = j.termID
        tab.lastAt = Date.now()
        const term = terms.get(tab.serverID)
        if (term) void termResize(j.termID, term.cols, term.rows)
      }
    })
    const offData = onTermData(({ id, b64 }) => {
      const tab = tabs.find((x) => x.termID === id)
      if (tab) output(tab, decodeOutput(b64))
    })
    return () => {
      offJob()
      offData()
    }
  })

  function output(tab: Tab, bytes: Uint8Array) {
    const term = terms.get(tab.serverID)
    if (term) term.write(bytes)
    else early.set(tab.serverID, [...(early.get(tab.serverID) ?? []), bytes])
    let dec = decoders.get(tab.serverID)
    if (!dec) decoders.set(tab.serverID, (dec = new TextDecoder()))
    const text = dec.decode(bytes, { stream: true })
    tab.tail = keepTail(tab.tail, text)
    tab.lastAt = Date.now()
    if (!tab.refused && sudoRefused(tab.tail)) tab.refused = true
  }

  function ready(tab: Tab, term: Terminal) {
    terms.set(tab.serverID, term)
    for (const b of early.get(tab.serverID) ?? []) term.write(b)
    early.delete(tab.serverID)
    term.onData((data) => {
      lastTyped = Date.now()
      if (hasRuntime && tab.termID && tab.state === 'running') void termWrite(tab.termID, data)
    })
    term.onResize(({ cols, rows }) => {
      if (hasRuntime && tab.termID && tab.state === 'running')
        void termResize(tab.termID, cols, rows)
    })
  }

  // A clock while it runs, to tell a quiet tab that waits.
  $effect(() => {
    if (!running) return
    const id = setInterval(() => (now = Date.now()), 500)
    return () => clearInterval(id)
  })

  // Bring a waiting tab forward, unless the user is typing somewhere.
  $effect(() => {
    void now
    const order = tabs.map((x) => x.serverID)
    untrack(() => {
      if (Date.now() - lastTyped < 3000) return
      const byID = new Map(tabs.map((x) => [x.serverID, x]))
      active = nextActive(order, (id) => isWaitingID(byID, id), active)
    })
  })
  const isWaitingID = (byID: Map<string, Tab>, id: string) => {
    const tab = byID.get(id)
    return tab ? isWaiting(tab) : false
  }

  onDestroy(() => {
    if (running) stopAll()
  })

  // The browser preview: a scripted run.
  function mock(items: FleetItem[]) {
    const steps = mockSteps(
      items.map((i) => i.serverID),
      items[0]?.command ?? '',
    )
    const enc = new TextEncoder()
    for (const s of steps)
      setTimeout(() => {
        const tab = tabs.find((x) => x.serverID === s.serverID)
        if (!tab) return
        if (s.state) tab.state = s.state
        if (s.exit !== undefined) tab.exit = s.exit
        if (s.out) output(tab, enc.encode(s.out))
      }, s.at)
  }

  const dot: Record<JobState, string> = {
    queued: 'var(--offline)',
    running: 'var(--accent)',
    done: 'var(--ok)',
    failed: 'var(--danger)',
    stopped: 'var(--warn)',
  }
  const stateText = (tab: Tab): string =>
    isWaiting(tab)
      ? r.state.waiting
      : tab.state === 'done' || tab.state === 'failed'
        ? r.exit(tab.exit)
        : r.state[tab.state]

  const commands = $derived(snippets.list.filter((s) => s.serverID === ''))
</script>

{#if tabs.length === 0}
  <div class="compose">
    <header class="head">
      <button class="btn" onclick={onback}>
        <Icon name="chevronLeft" size={12} strokeWidth={2.4} />{r.back}
      </button>
      <h3>{r.title}</h3>
    </header>
    <p class="lead">{r.lead}</p>

    <div class="pick">
      <label class="row all">
        <input
          type="checkbox"
          checked={online.length > 0 && online.every((s) => picked.has(s.id))}
          disabled={online.length === 0}
          onchange={(e) => {
            const on = e.currentTarget.checked
            for (const s of online) {
              if (on) picked.add(s.id)
              else picked.delete(s.id)
            }
          }}
        />
        <span>{r.allOnline(online.length)}</span>
      </label>
      <ul class="list">
        {#each servers.list as s (s.id)}
          {@const up = s.state === 'connected'}
          <li>
            <label class="row" class:off={!up}>
              <input
                type="checkbox"
                disabled={!up}
                checked={up && picked.has(s.id)}
                onchange={(e) => {
                  if (e.currentTarget.checked) picked.add(s.id)
                  else picked.delete(s.id)
                }}
              />
              <span class="name">{s.name}</span>
              <span class="addr mono">{s.user}@{s.host}</span>
              {#if !up}<span class="tag">{r.offline}</span>{/if}
            </label>
          </li>
        {/each}
      </ul>
    </div>

    <label class="cmd">
      <span>{r.command}</span>
      <input
        class="input mono"
        list="fleet-commands"
        spellcheck="false"
        autocomplete="off"
        placeholder="uptime"
        bind:value={command}
        onkeydown={(e) => e.key === 'Enter' && picked.size > 0 && runCommand()}
      />
      <datalist id="fleet-commands">
        {#each commands as c (c.id)}<option value={c.command}>{c.name}</option>{/each}
      </datalist>
    </label>
    {#if notice}<p class="notice" role="alert">{notice}</p>{/if}

    <div class="actions">
      <button
        class="btn"
        disabled={busy || picked.size === 0}
        title={r.updateHint}
        onclick={() => void runUpdate()}
      >
        <Icon name="package" size={13} />{r.update}
      </button>
      <span class="spacer"></span>
      <button
        class="btn btn-primary"
        disabled={busy || picked.size === 0 || command.trim() === ''}
        onclick={runCommand}>{r.runOn(picked.size)}</button
      >
    </div>
  </div>
{:else}
  <div class="run">
    <header class="head">
      <button class="btn" disabled={running} onclick={compose}>
        <Icon name="chevronLeft" size={12} strokeWidth={2.4} />{r.newCommand}
      </button>
      <span class="summary">{summary}</span>
      <span class="spacer"></span>
      {#if running}
        <button class="btn" onclick={stopAll}>{r.stopAll}</button>
      {:else if retry.length > 0}
        <button class="btn" onclick={again}>{r.again(retry.length)}</button>
      {/if}
    </header>

    <div class="tabs" role="tablist" aria-label={r.title}>
      {#each tabs as tab (tab.serverID)}
        {@const w = isWaiting(tab)}
        <button
          role="tab"
          class="tab"
          class:waiting={w}
          aria-selected={tab.serverID === active}
          onclick={() => (active = tab.serverID)}
        >
          <span class="dot" style:background={w ? 'var(--warn)' : dot[tab.state]}></span>
          <span class="tname">{nameOf(tab.serverID)}</span>
          <span class="tstate">{stateText(tab)}</span>
        </button>
      {/each}
    </div>

    {#if shown}
      <div class="bar">
        <code class="mono" title={shown.command}>{shown.command}</code>
      </div>
    {/if}

    <div class="screen" class:solid={settings.terminalGpu}>
      {#each tabs as tab (tab.serverID)}
        <XTerm active={tab.serverID === active} onready={(term) => ready(tab, term)} />
      {/each}
      {#if shown?.refused}
        <div class="note glass-dark" role="status">
          <Icon name="warning" size={13} />
          <span>{r.sudoRefused}</span>
          <button
            onclick={() =>
              shown &&
              openTerminalAt(
                shown.serverID,
                undefined,
                shown.command.replaceAll('sudo -n ', 'sudo '),
              )}>{r.openTerminal}</button
          >
        </div>
      {:else if shown?.error}
        <div class="note glass-dark" role="status">
          <Icon name="warning" size={13} />
          <span>{t.errors[shown.error as keyof typeof t.errors] ?? shown.error}</span>
        </div>
      {:else if shown && isWaiting(shown)}
        <div class="note glass-dark" role="status">
          <Icon name="terminal" size={13} />
          <span>{r.answerHere}</span>
        </div>
      {/if}
    </div>
  </div>
{/if}

<RunConfirm
  rows={pending
    ? pending.items.map((i) => ({ name: nameOf(i.serverID), command: i.command }))
    : null}
  check={pending?.check ?? { destructive: false, typed: false }}
  onclose={() => (pending = null)}
  onconfirm={() => void start()}
/>

<style>
  .compose {
    flex: 1;
    min-height: 0;
    overflow: auto;
    padding: 14px 18px;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .head {
    display: flex;
    align-items: center;
    gap: 10px;
    flex-shrink: 0;
  }

  h3 {
    margin: 0;
    font-size: var(--text-md);
  }

  .lead,
  .notice {
    margin: 0;
    font-size: var(--text-sm);
    color: var(--fg-muted);
  }

  .notice {
    color: var(--warn);
  }

  .list {
    list-style: none;
    margin: 0;
    padding: 4px;
    max-height: 260px;
    overflow: auto;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
  }

  .row {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 5px 6px;
    font-size: var(--text-sm);
  }

  .all {
    padding-inline: 10px;
  }

  .off {
    opacity: 0.55;
  }

  .name {
    font-weight: 600;
  }

  .addr {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--fg-muted);
  }

  .tag {
    font-size: var(--text-xs);
    color: var(--fg-subtle);
  }

  .cmd {
    display: flex;
    flex-direction: column;
    gap: 4px;
    font-size: var(--text-sm);
  }

  .actions {
    display: flex;
    gap: 8px;
  }

  .actions .btn {
    display: inline-flex;
    align-items: center;
    gap: 5px;
  }

  .spacer {
    flex: 1;
  }

  .run {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }

  .run .head {
    padding: 8px 12px;
    border-bottom: 1px solid var(--divider-soft);
  }

  .head .btn {
    display: inline-flex;
    align-items: center;
    gap: 3px;
  }

  .summary {
    font-size: var(--text-xs);
    color: var(--fg-muted);
  }

  .tabs {
    display: flex;
    gap: 4px;
    padding: 6px 10px;
    overflow-x: auto;
    flex-shrink: 0;
  }

  .tab {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 4px 10px;
    border: 0;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    font-size: var(--text-xs);
    white-space: nowrap;
  }

  .tab[aria-selected='true'] {
    background: var(--accent);
    color: var(--fg-on-accent);
  }

  .tab.waiting:not([aria-selected='true']) {
    box-shadow: inset 0 0 0 1.5px var(--warn);
  }

  .dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
  }

  .tname {
    font-weight: 600;
  }

  .tstate {
    opacity: 0.8;
  }

  .bar {
    padding: 0 12px 6px;
    flex-shrink: 0;
    font-size: var(--text-xs);
    color: var(--fg-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .bar code {
    -webkit-user-select: text;
    user-select: text;
  }

  .screen {
    position: relative;
    flex: 1;
    min-height: 0;
    margin: 0 10px 10px;
    border-radius: var(--control-radius);
    background: var(--app-terminal);
    overflow: hidden;
  }

  .solid {
    background: var(--term-bg-solid);
  }

  .note {
    position: absolute;
    left: 50%;
    bottom: 14px;
    translate: -50% 0;
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 12px;
    border-radius: var(--control-radius);
    font-size: 12px;
    max-width: calc(100% - 40px);
  }

  .note button {
    border: 0;
    border-radius: 5px;
    padding: 2px 9px;
    background: var(--tab-hover-dark);
    color: var(--fg-on-dark);
    font-size: 12px;
    white-space: nowrap;
  }
</style>
