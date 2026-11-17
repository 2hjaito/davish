<script lang="ts">
  import { SvelteSet } from 'svelte/reactivity'

  import { copyText } from '../../lib/api/clipboard'
  import type { AuditEntry } from '../../lib/api/events'
  import { t } from '../../lib/i18n'
  import { audit } from '../../lib/stores/audit.svelte'
  import { servers } from '../../lib/stores/servers.svelte'
  import { snippets } from '../../lib/stores/snippets.svelte'
  import Empty from '../../lib/ui/Empty.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import { openSnippets, openTerminalAt } from '../../lib/wm/intents.svelte'
  import TitleBar from '../../lib/wm/TitleBar.svelte'
  import type { AppProps } from '../types'
  import { defaultPick } from './exportScript'
  import ScriptDialog from './ScriptDialog.svelte'

  let { win }: AppProps = $props()

  let query = $state('')
  let onlyThis = $state(false)
  /** Only commands that changed a server; reads and polling are hidden. */
  let onlyChanges = $state(true)
  let copied = $state<number | null>(null)

  // Newest first, like Transfers.
  const rows = $derived.by(() => {
    const q = query.trim().toLowerCase()
    return audit.entries
      .filter(
        (e) =>
          (!onlyThis || e.serverID === servers.activeId) &&
          (!onlyChanges || e.mutating) &&
          (!q || e.command.toLowerCase().includes(q) || e.app.includes(q)),
      )
      .reverse()
  })

  function serverName(id: string) {
    return servers.list.find((s) => s.id === id)?.name ?? id
  }

  function clock(ms: number) {
    const d = new Date(ms)
    const p = (n: number) => String(n).padStart(2, '0')
    return `${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`
  }

  function duration(e: AuditEntry) {
    if (e.exit === -1 && e.app === 'logs') return t.audit.stream
    return e.durMs < 1000 ? `${e.durMs} ms` : `${(e.durMs / 1000).toFixed(1).replace('.', ',')} s`
  }

  // Typed at the prompt of the server it ran on, never run (docs/SECURITY.md).
  function rerun(e: AuditEntry) {
    if (e.serverID !== servers.activeId) servers.select(e.serverID)
    openTerminalAt(e.serverID, undefined, e.command)
  }

  function keep(e: AuditEntry) {
    snippets.draft = { command: e.command, serverID: e.serverID }
    openSnippets(servers.activeId || e.serverID)
  }

  // Choosing entries for a script (docs/adr/0022).
  let picking = $state(false)
  const picked = new SvelteSet<number>()
  let exporting = $state<AuditEntry[] | null>(null)

  function startPick() {
    picked.clear()
    for (const id of defaultPick(rows)) picked.add(id)
    picking = true
  }

  function toggle(id: number, on: boolean) {
    if (on) picked.add(id)
    else picked.delete(id)
  }

  async function copy(e: AuditEntry) {
    if (await copyText(e.command)) {
      copied = e.id
      setTimeout(() => copied === e.id && (copied = null), 1500)
    }
  }
</script>

<TitleBar>
  <span class="title">{win.title}</span>
  <span class="spacer"></span>
</TitleBar>

<div class="body">
  <div class="bar">
    <label class="search">
      <Icon name="search" size={13} strokeWidth={2.2} />
      <input
        type="text"
        placeholder={t.audit.search}
        aria-label={t.audit.search}
        bind:value={query}
        onkeydown={(e) => e.key === 'Escape' && (query = '')}
      />
    </label>
    <label class="only">
      <input type="checkbox" class="switch" role="switch" bind:checked={onlyChanges} />
      {t.audit.onlyChanges}
    </label>
    <label class="only">
      <input type="checkbox" class="switch" role="switch" bind:checked={onlyThis} />
      {t.audit.onlyThis}
    </label>
    {#if !picking}
      <button class="btn export" disabled={rows.length === 0} onclick={startPick}>
        <Icon name="download" size={13} strokeWidth={2.2} />{t.audit.export.start}
      </button>
    {/if}
  </div>

  {#if audit.entries.length === 0}
    <Empty icon="list" title={t.audit.empty} hint={t.audit.emptyHint} />
  {:else if rows.length === 0}
    <Empty
      icon="search"
      title={t.audit.noMatch}
      hint={onlyChanges && !query ? t.audit.noChanges : undefined}
    />
  {:else}
    <ul class="rows">
      {#each rows as e (e.id)}
        <li class="row" class:picked={picking && picked.has(e.id)}>
          {#if picking}
            <input
              type="checkbox"
              aria-label={t.audit.export.pick}
              checked={picked.has(e.id)}
              onchange={(ev) => toggle(e.id, ev.currentTarget.checked)}
            />
          {/if}
          <span
            class="exit"
            class:ok={e.exit === 0}
            class:bad={e.exit > 0}
            title={e.exit === -1 ? t.audit.noExit : t.audit.exitCode(e.exit)}
          >
            {#if e.exit === 0}<Icon name="check" size={12} strokeWidth={2.8} />
            {:else if e.exit > 0}{e.exit}
            {:else}<Icon name="minus" size={12} strokeWidth={2.8} />{/if}
          </span>
          <div class="info">
            <code class="cmd" title={e.command}>{e.command}</code>
            <span class="meta">
              {clock(e.time)} · {serverName(e.serverID)} · {t.audit.apps[e.app] ?? e.app} · {duration(
                e,
              )}
            </span>
          </div>
          <button
            class="copy"
            aria-label={t.audit.rerun}
            title={t.audit.rerun}
            disabled={servers.list.find((s) => s.id === e.serverID)?.state !== 'connected'}
            onclick={() => rerun(e)}
          >
            <Icon name="terminal" size={13} strokeWidth={2.2} />
          </button>
          <button
            class="copy"
            aria-label={t.audit.save}
            title={t.audit.save}
            onclick={() => keep(e)}
          >
            <Icon name="pin" size={13} strokeWidth={2.2} />
          </button>
          <button
            class="copy"
            class:done={copied === e.id}
            aria-label={t.audit.copy}
            title={copied === e.id ? t.audit.copied : t.audit.copy}
            onclick={() => void copy(e)}
          >
            <Icon name={copied === e.id ? 'check' : 'copy'} size={13} strokeWidth={2.2} />
          </button>
        </li>
      {/each}
    </ul>
  {/if}

  {#if picking}
    <div class="pickbar">
      <span>{t.audit.export.picked(picked.size)}</span>
      <button class="link" onclick={() => rows.forEach((e) => picked.add(e.id))}
        >{t.audit.export.all}</button
      >
      <button class="link" onclick={() => picked.clear()}>{t.audit.export.none}</button>
      <span class="grow"></span>
      <button class="btn" onclick={() => (picking = false)}>{t.common.cancel}</button>
      <button
        class="btn btn-primary"
        disabled={picked.size === 0}
        onclick={() => (exporting = audit.entries.filter((e) => picked.has(e.id)))}
        >{t.audit.export.show}</button
      >
    </div>
  {:else}
    <p class="footer">{t.audit.footer(audit.entries.length)}</p>
  {/if}
</div>

<ScriptDialog entries={exporting} onclose={() => (exporting = null)} />

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
    min-height: 0;
    display: flex;
    flex-direction: column;
    background: var(--content-bg);
  }

  .bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 8px 12px;
    border-bottom: 1px solid var(--divider-soft);
  }

  .search {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 6px;
    background: var(--fill-soft);
    border-radius: 8px;
    padding: 4px 9px;
    font-size: var(--text-sm);
    color: var(--fg-muted);
  }

  .search input {
    flex: 1;
    min-width: 0;
    border: 0;
    background: transparent;
    outline: none;
    color: var(--fg);
  }

  .only {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: var(--text-sm);
    color: var(--fg-muted);
    white-space: nowrap;
  }

  .rows {
    flex: 1;
    min-height: 0;
    margin: 0;
    padding: 6px 8px;
    list-style: none;
    overflow-y: auto;
  }

  .row {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 6px 8px;
    border-radius: var(--control-radius);
  }

  .row:nth-child(even) {
    background: var(--row-stripe);
  }

  .exit {
    width: 22px;
    height: 22px;
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 50%;
    background: var(--fill-soft);
    color: var(--fg-subtle);
    font-size: var(--text-xs);
    font-weight: 600;
    font-variant-numeric: tabular-nums;
  }

  .exit.ok {
    color: var(--ok);
  }

  .exit.bad {
    color: var(--danger);
  }

  .info {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .cmd {
    font-family: var(--font-mono);
    font-size: var(--text-sm);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    user-select: text;
  }

  .meta {
    font-size: var(--text-xs);
    color: var(--fg-subtle);
    font-variant-numeric: tabular-nums;
  }

  .copy {
    display: flex;
    padding: 5px;
    border: 0;
    border-radius: 50%;
    background: none;
    color: var(--fg-muted);
    opacity: 0;
    transition: opacity var(--dur-fast) var(--ease-out);
  }

  .row:hover .copy:disabled {
    opacity: 0.35;
  }

  .row:hover .copy,
  .copy:focus-visible,
  .copy.done {
    opacity: 1;
  }

  .copy:hover {
    background: var(--fill-soft);
  }

  .copy.done {
    color: var(--ok);
  }

  .export {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    white-space: nowrap;
  }

  .row.picked {
    background: var(--selection-inactive);
  }

  .pickbar {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 6px 12px;
    border-top: 1px solid var(--divider-soft);
    font-size: var(--text-sm);
  }

  .pickbar .grow {
    flex: 1;
  }

  .link {
    border: 0;
    background: none;
    padding: 0;
    color: var(--accent);
    font-size: var(--text-sm);
  }

  .footer {
    margin: 0;
    padding: 6px 14px;
    border-top: 1px solid var(--divider-soft);
    font-size: var(--text-xs);
    color: var(--fg-muted);
  }
</style>
