<script lang="ts">
  // Searching a log source's whole history on the server, beyond the 200
  // lines a stream starts with. For a file, a hit opens a page of the file
  // around it, read with sed, so a file of several GB is never loaded.
  import { untrack } from 'svelte'

  import { toUserError } from '../../lib/api/errors'
  import {
    type LogLine,
    type LogPage,
    type LogSearchResult,
    type LogSource,
    logsAround,
    logsCancelSearch,
    logsSearch,
  } from '../../lib/api/logs'
  import { hasRuntime } from '../../lib/api/runtime'
  import { t } from '../../lib/i18n'
  import Empty from '../../lib/ui/Empty.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import { highlight, searchToken } from '../../lib/search'
  import { aroundStart, mockAround, mockLogSearch, PAGE } from './history'

  let {
    serverID,
    source,
    query,
    matchCase,
    demoLines,
    onclose,
  }: {
    serverID: string
    source: LogSource
    query: string
    matchCase: boolean
    /** The lines on screen, searched instead of the server in the browser preview. */
    demoLines: readonly string[]
    onclose: () => void
  } = $props()

  let result = $state.raw<LogSearchResult | null>(null)
  let error = $state('')
  let token = ''
  /** The page of the file being read around a hit. */
  let page = $state.raw<(LogPage & { from: number; target: number }) | null>(null)
  let pageError = $state('')
  let paging = $state(false)

  const running = $derived(result === null && error === '')

  async function run() {
    cancel()
    result = null
    error = ''
    page = null
    const mine = searchToken()
    token = mine
    try {
      const r = hasRuntime
        ? await logsSearch(serverID, source, { query, matchCase }, mine)
        : mockLogSearch(demoLines, query, matchCase)
      if (token === mine) result = r
    } catch (e) {
      if (token === mine) error = toUserError(e).message
    } finally {
      if (token === mine) token = ''
    }
  }

  function cancel() {
    if (token && hasRuntime) void logsCancelSearch(token)
    token = ''
  }

  function stop() {
    cancel()
    error = t.logs.history.stopped
  }

  // A new query or source searches again; leaving stops the search.
  // Lines streaming in (demoLines) must not start it again.
  $effect(() => {
    void [serverID, source, query, matchCase]
    untrack(() => void run())
    return cancel
  })

  async function read(from: number, target: number) {
    paging = true
    pageError = ''
    try {
      const p = hasRuntime
        ? await logsAround(serverID, source.target, from, PAGE)
        : mockAround(demoLines, from, PAGE)
      page = { ...p, from, target }
    } catch (e) {
      pageError = toUserError(e).message
    } finally {
      paging = false
    }
  }

  function openHit(l: LogLine) {
    if (source.kind !== 'file' || l.number < 1) return
    void read(aroundStart(l.number), l.number)
  }
</script>

<div class="history">
  <div class="bar" role="status">
    {#if page}
      <button class="btn" onclick={() => (page = null)}>
        <Icon name="chevronLeft" size={12} strokeWidth={2.4} />
        {t.logs.history.backToResults}
      </button>
      <span>{t.logs.history.pageOf(page.from, page.from + page.lines.length - 1)}</span>
      <span class="spacer"></span>
      <button
        class="btn"
        disabled={paging || page.from <= 1}
        onclick={() => page && void read(Math.max(1, page.from - PAGE), page.target)}
        >{t.logs.history.earlier}</button
      >
      <button
        class="btn"
        disabled={paging || page.eof}
        onclick={() => page && void read(page.from + PAGE, page.target)}
        >{t.logs.history.later}</button
      >
    {:else}
      {#if running}
        <span>{t.logs.history.searching}</span>
        <button class="btn" onclick={stop}>{t.logs.history.stop}</button>
      {:else if error}
        <Icon name="warning" size={12} strokeWidth={2.2} />
        <span>{error}</span>
      {:else if result}
        <span>{t.logs.history.found(result.lines.length)}</span>
        {#if result.truncated}<span class="note">· {t.logs.history.truncated}</span>{/if}
        {#if result.timedOut}<span class="note">· {t.logs.history.timedOut}</span>{/if}
        {#if result.restricted}<span class="note">· {t.logs.history.restricted}</span>{/if}
      {/if}
      <span class="spacer"></span>
      {#if result?.command}<code class="command" title={result.command}>{result.command}</code>{/if}
    {/if}
    <button class="btn" onclick={onclose}>{t.logs.history.close}</button>
  </div>

  {#if page}
    {#if pageError}
      <Empty icon="warning" title={pageError} />
    {:else}
      <div class="lines">
        {#each page.lines as l (l.number)}
          <div class="line" class:target={l.number === page.target}>
            <span class="num">{l.number}</span>
            <span class="text">{l.text}</span>
          </div>
        {/each}
      </div>
    {/if}
  {:else if result && result.lines.length === 0}
    <Empty icon="search" title={t.logs.history.none} />
  {:else if result}
    <div class="lines">
      {#each result.lines as l, i (`${l.number}:${i}`)}
        {@const h = highlight(l.text, query, matchCase)}
        {#if source.kind === 'file' && l.number > 0}
          <button
            class="line hit"
            title={t.logs.history.openAround(l.number)}
            onclick={() => openHit(l)}
          >
            <span class="num">{l.number}</span>
            <span class="text"
              >{h.before}{#if h.match}<mark>{h.match}</mark>{/if}{h.after}</span
            >
          </button>
        {:else}
          <div class="line">
            <span class="text"
              >{h.before}{#if h.match}<mark>{h.match}</mark>{/if}{h.after}</span
            >
          </div>
        {/if}
      {/each}
    </div>
  {/if}
</div>

<style>
  .history {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }

  .bar {
    flex-shrink: 0;
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 6px;
    padding: 5px 10px;
    border-bottom: 1px solid var(--divider-dark);
    font-size: var(--text-xs);
    color: var(--fg-on-dark-muted);
    min-width: 0;
  }

  .spacer {
    flex: 1;
  }

  .note {
    color: var(--warn-on-dark);
  }

  .command {
    min-width: 0;
    max-width: 45%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-family: var(--font-mono);
    font-size: var(--text-xxs);
    -webkit-user-select: text;
    user-select: text;
  }

  .btn {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    padding: 2px 9px;
    border: 0;
    border-radius: 6px;
    background: var(--tab-hover-dark);
    color: var(--fg-on-dark);
    font-size: var(--text-xs);
    white-space: nowrap;
  }

  .btn:disabled {
    opacity: 0.4;
  }

  .lines {
    flex: 1;
    min-height: 0;
    overflow: auto;
    padding: 6px 0 10px;
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    color: var(--fg-on-dark);
    -webkit-user-select: text;
    user-select: text;
  }

  .line {
    display: flex;
    gap: 12px;
    width: max-content;
    min-width: 100%;
    padding: 0 14px;
    line-height: 18px;
    border: 0;
    background: none;
    color: inherit;
    font: inherit;
    text-align: left;
    white-space: pre;
  }

  .line.hit {
    cursor: pointer;
  }

  .line.hit:hover,
  .line.target {
    background: var(--tab-hover-dark);
  }

  .line.target {
    box-shadow: inset 2px 0 0 var(--accent);
  }

  .num {
    flex-shrink: 0;
    min-width: 6ch;
    text-align: right;
    color: var(--fg-on-dark-muted);
    -webkit-user-select: none;
    user-select: none;
  }

  mark {
    background: color-mix(in srgb, var(--accent) 45%, transparent);
    color: inherit;
    border-radius: 2px;
  }
</style>
