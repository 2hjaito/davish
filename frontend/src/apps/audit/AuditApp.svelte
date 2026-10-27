<script lang="ts">
  import { copyText } from '../../lib/api/clipboard'
  import type { AuditEntry } from '../../lib/api/events'
  import { t } from '../../lib/i18n/vi'
  import { audit } from '../../lib/stores/audit.svelte'
  import { servers } from '../../lib/stores/servers.svelte'
  import Empty from '../../lib/ui/Empty.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import TitleBar from '../../lib/wm/TitleBar.svelte'
  import type { AppProps } from '../types'

  let { win }: AppProps = $props()

  let query = $state('')
  let onlyThis = $state(false)
  let copied = $state<number | null>(null)

  // Newest first, like Transfers.
  const rows = $derived.by(() => {
    const q = query.trim().toLowerCase()
    return audit.entries
      .filter(
        (e) =>
          (!onlyThis || e.serverID === servers.activeId) &&
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
      <input type="checkbox" class="switch" role="switch" bind:checked={onlyThis} />
      {t.audit.onlyThis}
    </label>
  </div>

  {#if audit.entries.length === 0}
    <Empty icon="list" title={t.audit.empty} hint={t.audit.emptyHint} />
  {:else if rows.length === 0}
    <Empty icon="search" title={t.audit.noMatch} />
  {:else}
    <ul class="rows">
      {#each rows as e (e.id)}
        <li class="row">
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

  <p class="footer">{t.audit.footer(audit.entries.length)}</p>
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

  .footer {
    margin: 0;
    padding: 6px 14px;
    border-top: 1px solid var(--divider-soft);
    font-size: var(--text-xs);
    color: var(--fg-muted);
  }
</style>
