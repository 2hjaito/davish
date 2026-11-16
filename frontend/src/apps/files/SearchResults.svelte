<script lang="ts">
  // Hits of a server-side search, in place of the folder listing. Up to
  // 500 rows, so no virtual list.
  import type { SearchHit, SearchMode, SearchResult } from '../../lib/api/files'
  import { t } from '../../lib/i18n'
  import Empty from '../../lib/ui/Empty.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import { describe, highlight } from './search'

  let {
    from,
    query,
    mode,
    matchCase,
    result,
    running,
    error,
    onopen,
    onreveal,
    onstop,
  }: {
    /** The folder the search started in. */
    from: string
    query: string
    mode: SearchMode
    matchCase: boolean
    result: SearchResult | null
    running: boolean
    error: string
    /** Open the hit: a folder, or a file in the Editor (at its line). */
    onopen: (hit: SearchHit) => void
    /** Show the hit in its folder. */
    onreveal: (hit: SearchHit) => void
    onstop: () => void
  } = $props()

  let selected = $state(-1)
  const hits = $derived(result?.hits ?? [])

  function onkeydown(e: KeyboardEvent) {
    if (hits.length === 0) return
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      e.preventDefault()
      const step = e.key === 'ArrowDown' ? 1 : -1
      selected = Math.max(0, Math.min(hits.length - 1, selected + step))
      document.getElementById(rowID(selected))?.focus()
    }
  }

  const rowID = (i: number) => `search-hit-${i}`
</script>

<div class="results">
  <div class="status" role="status">
    {#if running}
      <span class="spin"><Icon name="refresh" size={12} strokeWidth={2.2} /></span>
      <span>{mode === 'name' ? t.files.searchingNames(from) : t.files.searchingContent(from)}</span>
      <button class="stop" onclick={onstop}>{t.files.searchStop}</button>
    {:else if error}
      <Icon name="warning" size={12} strokeWidth={2.2} />
      <span>{error}</span>
    {:else if result}
      <span>{t.files.searchFound(hits.length)}</span>
      {#if result.truncated}<span class="note">· {t.files.searchTruncated}</span>{/if}
      {#if result.timedOut}<span class="note">· {t.files.searchTimedOut}</span>{/if}
      {#if result.denied > 0}<span class="note">· {t.files.searchDenied(result.denied)}</span>{/if}
    {/if}
    {#if result?.command && !running}
      <code class="command" title={result.command}>{result.command}</code>
    {/if}
  </div>

  {#if !running && result && hits.length === 0}
    <Empty icon="search" title={t.files.searchNone} hint={t.files.searchNoneHint} />
  {:else}
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="scroller" {onkeydown}>
      {#each hits as hit, i (`${hit.path}:${hit.line ?? 0}:${i}`)}
        {@const d = describe(hit, from)}
        <div class="row" class:odd={i % 2 === 1} class:selected={selected === i}>
          <button
            id={rowID(i)}
            class="open"
            title={hit.path}
            onfocus={() => (selected = i)}
            onclick={() => (selected = i)}
            ondblclick={() => onopen(hit)}
            onkeydown={(e) => e.key === 'Enter' && onopen(hit)}
          >
            <Icon name={mode === 'content' ? 'file' : 'search'} size={14} strokeWidth={1.8} />
            <span class="name">{d.name}</span>
            {#if hit.line}<span class="line">:{hit.line}</span>{/if}
            <span class="where">{d.where}</span>
            {#if hit.text !== undefined}
              {@const h = highlight(hit.text, query, matchCase)}
              <span class="text"
                >{h.before}{#if h.match}<mark>{h.match}</mark>{/if}{h.after}</span
              >
            {/if}
          </button>
          <button
            class="reveal"
            aria-label={t.files.searchReveal}
            title={t.files.searchReveal}
            onclick={() => onreveal(hit)}
          >
            <Icon name="folder" size={13} strokeWidth={2} />
          </button>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .results {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    font-size: var(--text-md);
  }

  .status {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    gap: 6px;
    min-height: 28px;
    padding: 0 18px;
    border-bottom: 1px solid var(--divider-soft);
    font-size: var(--text-xs);
    color: var(--fg-muted);
    min-width: 0;
  }

  .note {
    color: var(--warn);
  }

  .command {
    margin-left: auto;
    min-width: 0;
    max-width: 50%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-family: var(--font-mono);
    font-size: var(--text-xxs);
    color: var(--fg-subtle);
    user-select: text;
    -webkit-user-select: text;
  }

  .stop {
    border: 0;
    border-radius: var(--control-radius);
    padding: 2px 8px;
    background: var(--fill-soft);
    color: var(--fg);
    font-size: var(--text-xs);
  }

  .spin {
    display: inline-flex;
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .spin {
      animation: none;
    }
  }

  .scroller {
    flex: 1;
    min-height: 0;
    padding: 4px 8px 6px;
    overflow-y: auto;
    scrollbar-gutter: stable;
  }

  .row {
    display: flex;
    align-items: center;
    border-radius: var(--control-radius);
  }

  .row.odd {
    background: var(--row-stripe);
  }

  .row:hover {
    background: var(--row-hover);
  }

  .row.selected {
    background: var(--selection-inactive);
  }

  .row.selected:focus-within {
    background: var(--accent);
    color: var(--fg-on-accent);
  }

  .open {
    flex: 1;
    min-width: 0;
    height: 30px;
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 0 10px;
    border: 0;
    background: transparent;
    color: inherit;
    text-align: left;
    outline: none;
  }

  .name {
    flex-shrink: 0;
    max-width: 40%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .line {
    flex-shrink: 0;
    margin-left: -6px;
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    opacity: 0.7;
  }

  .where {
    flex-shrink: 1;
    min-width: 40px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: var(--text-xs);
    opacity: 0.65;
  }

  .text {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: pre;
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    opacity: 0.85;
  }

  mark {
    background: var(--accent-soft);
    color: inherit;
    border-radius: 2px;
  }

  .row.selected:focus-within mark {
    background: var(--glass-highlight);
  }

  .reveal {
    flex-shrink: 0;
    display: inline-flex;
    padding: 4px 8px;
    border: 0;
    background: transparent;
    color: inherit;
    opacity: 0;
  }

  .row:hover .reveal,
  .row:focus-within .reveal {
    opacity: 0.8;
  }
</style>
