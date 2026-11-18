<script lang="ts">
  // One SQLite table, read-only, a page of 100 rows at a time (ADR 0024).
  import { untrack } from 'svelte'

  import type { SQLiteCell, SQLiteQuery, SQLiteRows } from '../../lib/api/database'
  import { toUserError } from '../../lib/api/errors'
  import { formatBytes } from '../../lib/format'
  import { t } from '../../lib/i18n'
  import Icon from '../../lib/ui/Icon.svelte'
  import { backend } from './backend'

  interface Props {
    serverID: string
    file: string
    table: string
    /** Rows counted when the file was read, for "of n". */
    total: number
    onback: () => void
  }

  let { serverID, file, table, total, onback }: Props = $props()
  const d = t.database.sqlite

  let page = $state.raw<SQLiteRows | null>(null)
  let offset = $state(0)
  let loading = $state(false)
  let error = $state('')
  let seq = 0
  // Sort and filter, built into SQL by the backend; no SQL is typed.
  let sort = $state<{ col: string; desc: boolean } | null>(null)
  let column = $state('')
  let contains = $state('')
  let applied = $state.raw<SQLiteQuery>({})
  const filtered = $derived(!!applied.contains)

  async function load(at: number) {
    const n = ++seq
    loading = true
    error = ''
    try {
      const p = await backend.sqliteRows(serverID, file, table, at, applied)
      if (n !== seq) return
      page = p
      offset = at
    } catch (e) {
      if (n === seq) error = toUserError(e).message
    } finally {
      if (n === seq) loading = false
    }
  }

  $effect(() => {
    void table
    untrack(() => {
      sort = null
      column = ''
      contains = ''
      applied = {}
      void load(0)
    })
  })

  function query(): SQLiteQuery {
    const q: SQLiteQuery = {}
    if (sort) {
      q.sort = sort.col
      q.desc = sort.desc
    }
    const col = column || page?.columns[0]?.name || ''
    if (contains.trim() && col) {
      q.column = col
      q.contains = contains.trim()
    }
    return q
  }

  function apply() {
    applied = query()
    void load(0)
  }

  // A header cycles: ascending, descending, unsorted.
  function toggleSort(col: string) {
    if (sort?.col !== col) sort = { col, desc: false }
    else if (!sort.desc) sort = { col, desc: true }
    else sort = null
    apply()
  }

  function show(c: SQLiteCell): string {
    if (c.k === 'null') return 'NULL'
    if (c.k === 'blob') return d.blob(formatBytes(c.size ?? 0))
    return c.cut ? `${c.v ?? ''}…` : (c.v ?? '')
  }
</script>

<div class="view">
  <div class="bar">
    <button class="btn" onclick={onback} aria-label={d.back}>
      <Icon name="chevronLeft" size={13} />
      {d.back}
    </button>
    <span class="name mono" title={file}>{table}</span>
    <span class="spacer"></span>
    {#if page}
      <small class="muted">
        {page.rows.length === 0
          ? d.noRows
          : filtered
            ? d.rangeFiltered(offset + 1, offset + page.rows.length)
            : d.range(
                offset + 1,
                offset + page.rows.length,
                Math.max(total, offset + page.rows.length),
              )}
      </small>
    {/if}
    <button
      class="btn"
      aria-label={d.prev}
      title={d.prev}
      disabled={loading || offset === 0}
      onclick={() => void load(Math.max(0, offset - 100))}
    >
      <Icon name="chevronLeft" size={13} />
    </button>
    <button
      class="btn"
      aria-label={d.next}
      title={d.next}
      disabled={loading || !page?.more}
      onclick={() => void load(offset + 100)}
    >
      <Icon name="chevronRight" size={13} />
    </button>
  </div>

  {#if error}
    <p class="error" role="alert">{error}</p>
  {:else if !page}
    <p class="muted pad">{t.database.loading}</p>
  {:else}
    <form
      class="filter"
      onsubmit={(e) => {
        e.preventDefault()
        apply()
      }}
    >
      <Icon name="search" size={13} />
      <select class="input" bind:value={column} aria-label={d.filterColumn}>
        {#each page.columns as c, i (i)}<option value={i === 0 ? '' : c.name}>{c.name}</option
          >{/each}
      </select>
      <input
        class="input"
        bind:value={contains}
        placeholder={d.filterContains}
        aria-label={d.filterContains}
        maxlength={1024}
      />
      <button class="btn" type="submit" disabled={loading}>{d.filter}</button>
      {#if filtered}
        <button
          class="btn"
          type="button"
          onclick={() => {
            contains = ''
            apply()
          }}>{d.clearFilter}</button
        >
      {/if}
    </form>
    {#if page.allColumns > page.columns.length}
      <p class="muted pad small">{d.someColumns(page.columns.length, page.allColumns)}</p>
    {/if}
    <div class="grid" class:dim={loading}>
      <table>
        <thead>
          <tr>
            <th class="num">#</th>
            {#each page.columns as c, i (i)}
              <th
                title={c.type}
                aria-sort={sort?.col === c.name
                  ? sort.desc
                    ? 'descending'
                    : 'ascending'
                  : undefined}
              >
                <button class="sort" onclick={() => toggleSort(c.name)} title={d.sortBy(c.name)}>
                  {#if c.pk}<Icon name="key" size={11} />{/if}
                  {c.name}
                  {#if c.type}<small>{c.type}</small>{/if}
                  {#if sort?.col === c.name}<Icon
                      name={sort.desc ? 'arrowDown' : 'arrowUp'}
                      size={11}
                    />{/if}
                </button>
              </th>
            {/each}
          </tr>
        </thead>
        <tbody>
          {#each page.rows as row, r (offset + r)}
            <tr>
              <td class="num">{offset + r + 1}</td>
              {#each row as c, i (i)}
                <td
                  class:null={c.k === 'null' || c.k === 'blob'}
                  class:right={c.k === 'num'}
                  title={c.k === 'text' ? c.v + (c.cut ? '…' : '') : undefined}>{show(c)}</td
                >
              {/each}
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
    <p class="muted pad small">{d.readOnly}</p>
  {/if}
</div>

<style>
  .view {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    background: var(--content-bg);
  }

  .bar {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 8px 12px;
    border-bottom: 1px solid var(--divider-soft);
  }

  .bar .btn {
    display: inline-flex;
    align-items: center;
    gap: 4px;
  }

  .name {
    font-weight: 600;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .spacer {
    flex: 1;
  }

  .muted {
    color: var(--fg-muted);
  }

  .small {
    font-size: var(--text-sm);
  }

  .pad {
    margin: 6px 12px;
  }

  .error {
    margin: 12px;
    color: var(--danger);
  }

  .filter {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 6px 12px;
    color: var(--fg-muted);
  }

  .filter select {
    max-width: 160px;
  }

  .filter input {
    flex: 1;
    min-width: 0;
    max-width: 320px;
  }

  .sort {
    border: 0;
    background: none;
    padding: 0;
    font: inherit;
    color: inherit;
    display: inline-flex;
    align-items: center;
    gap: 4px;
  }

  .sort:hover {
    color: var(--fg);
  }

  .grid {
    flex: 1;
    min-height: 0;
    overflow: auto;
  }

  .grid.dim {
    opacity: 0.6;
  }

  table {
    border-collapse: collapse;
    font-family: var(--font-mono);
    font-size: var(--text-sm);
  }

  th,
  td {
    padding: 4px 10px;
    text-align: left;
    white-space: nowrap;
    max-width: 320px;
    overflow: hidden;
    text-overflow: ellipsis;
    border-bottom: 1px solid var(--divider-soft);
    -webkit-user-select: text;
    user-select: text;
  }

  th {
    position: sticky;
    top: 0;
    background: var(--card-bg);
    font-weight: 600;
    color: var(--fg-label);
  }

  th small {
    font-weight: 400;
    color: var(--fg-muted);
    margin-left: 4px;
  }

  th :global(svg) {
    color: var(--warn);
    vertical-align: -1px;
  }

  .num {
    color: var(--fg-muted);
    text-align: right;
  }

  .right {
    text-align: right;
  }

  .null {
    color: var(--fg-muted);
    font-style: italic;
  }
</style>
