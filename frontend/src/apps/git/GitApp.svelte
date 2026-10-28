<script lang="ts">
  import { untrack } from 'svelte'

  import { toUserError } from '../../lib/api/errors'
  import {
    LOG_PAGE,
    type Change,
    type Commit,
    type Diff,
    type Repo,
    type Status,
  } from '../../lib/api/git'
  import { t } from '../../lib/i18n/vi'
  import { load, save } from '../../lib/stores/persist'
  import { servers } from '../../lib/stores/servers.svelte'
  import Empty from '../../lib/ui/Empty.svelte'
  import FilterShell, { type FilterItem } from '../../lib/ui/FilterShell.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import StatusList, { type StatusItem } from '../../lib/ui/StatusList.svelte'
  import { getWindowContext } from '../../lib/wm/context'
  import type { AppProps } from '../types'
  import { backend } from './backend'
  import FoldersDialog from './FoldersDialog.svelte'
  import RepoInspector, { type Selected } from './RepoInspector.svelte'

  type Tab = 'changes' | 'history'

  let { win }: AppProps = $props()
  const ctx = getWindowContext()

  // Folders and the open repository are kept per server.
  const saved = load('git', {
    tab: 'changes' as Tab,
    folders: {} as Record<string, string[]>,
    repo: {} as Record<string, string>,
  })
  let tab = $state<Tab>(saved.tab === 'history' ? 'history' : 'changes')
  let folders = $state<Record<string, string[]>>(saved.folders)
  let lastRepo = $state<Record<string, string>>(saved.repo)
  let repos = $state.raw<Repo[] | null>(null)
  let findError = $state('')
  let finding = $state(false)
  let repo = $state('')
  let status = $state.raw<Status | null>(null)
  let commits = $state.raw<Commit[] | null>(null)
  let more = $state(false)
  let error = $state('')
  let query = $state('')
  let selected = $state<string | null>(null)
  let diff = $state.raw<Diff | null>(null)
  let diffError = $state('')
  let editingFolders = $state(false)

  $effect(() => save('git', { tab, folders, repo: lastRepo }))

  const server = $derived(servers.list.find((s) => s.id === win.serverID))
  const connected = $derived(server?.state === 'connected')
  const extra = $derived(folders[win.serverID] ?? [])

  // Two repositories may share a name; the parent folder tells them apart.
  const filters = $derived<FilterItem[]>(
    (repos ?? []).map((r) => {
      const twin = (repos ?? []).some((o) => o !== r && o.name === r.name)
      const parent = r.path.slice(0, r.path.length - r.name.length - 1)
      return {
        id: r.path,
        label: twin ? `${r.name} — ${parent.split('/').pop() || '/'}` : r.name,
        icon: 'gitBranch',
        count: r.path === repo && status ? status.changes.length || undefined : undefined,
      }
    }),
  )

  function changeNote(c: Change): string {
    if (c.conflict) return t.git.conflict
    const letter = c.worktree !== '.' ? c.worktree : c.index
    const state = t.git.states[letter] ?? letter
    return c.index !== '.' && c.index !== '?' ? `${state} · ${t.git.staged}` : state
  }

  const dateTime = (s: number) => new Date(s * 1000).toLocaleString('vi-VN')
  const date = (s: number) => new Date(s * 1000).toLocaleDateString('vi-VN')

  type Row = StatusItem & { sel: Selected; text: string }

  const rows = $derived.by<Row[]>(() => {
    if (tab === 'changes')
      return (status?.changes ?? []).map((c) => ({
        key: `f:${c.path}`,
        name: c.path,
        health: c.conflict ? 'failed' : c.index === '?' ? 'off' : 'busy',
        note: changeNote(c),
        title: c.origPath ? t.git.renamedFrom(c.origPath) : c.path,
        sel: { kind: 'change', change: c },
        text: `${c.path} ${c.origPath}`,
      }))
    return (commits ?? []).map((c) => ({
      key: `c:${c.hash}`,
      name: c.subject,
      health: 'ok',
      note: date(c.time),
      title: `${c.short} · ${c.author} · ${dateTime(c.time)}`,
      sel: { kind: 'commit', commit: c },
      text: `${c.subject} ${c.hash} ${c.author}`,
    }))
  })

  const shown = $derived.by(() => {
    const q = query.trim().toLowerCase()
    return rows.filter((r) => !q || r.text.toLowerCase().includes(q))
  })
  const current = $derived(shown.find((r) => r.key === selected)?.sel ?? null)

  const footer = $derived.by(() => {
    if (!status) return ''
    const where = status.branch || t.git.detached
    return tab === 'changes'
      ? `${where} · ${t.git.count(status.changes.length)}`
      : `${where} · ${t.git.commits(commits?.length ?? 0)}`
  })

  async function findRepos() {
    if (finding || !connected) return
    finding = true
    try {
      const found = await backend.find(win.serverID, extra)
      repos = found
      findError = ''
      if (!found.some((r) => r.path === repo)) {
        const last = lastRepo[win.serverID]
        repo = found.find((r) => r.path === last)?.path ?? found[0]?.path ?? ''
      }
    } catch (e) {
      findError = toUserError(e).message
    } finally {
      finding = false
    }
  }

  // Answers for a repository no longer open are dropped.
  async function loadStatus(dir: string) {
    try {
      const st = await backend.status(win.serverID, dir)
      if (dir !== repo) return
      status = st
      error = ''
    } catch (e) {
      if (dir !== repo) return
      status = null
      error = toUserError(e).message
    }
  }

  async function loadLog(dir: string, skip: number) {
    try {
      const page = await backend.log(win.serverID, dir, skip)
      if (dir !== repo) return
      commits = skip === 0 ? page : [...(commits ?? []), ...page]
      more = page.length === LOG_PAGE
    } catch (e) {
      if (dir !== repo) return
      error = toUserError(e).message
    }
  }

  function refresh() {
    void findRepos()
    if (!repo) return
    void loadStatus(repo)
    if (tab === 'history') void loadLog(repo, 0)
  }

  // Opening a repository starts it fresh.
  $effect(() => {
    const dir = repo
    untrack(() => {
      status = null
      commits = null
      selected = null
      error = ''
      if (!dir) return
      lastRepo = { ...lastRepo, [win.serverID]: dir }
      void loadStatus(dir)
    })
  })

  $effect(() => {
    if (tab === 'history' && repo && commits === null) untrack(() => void loadLog(repo, 0))
  })

  // The diff of what is selected, dropped when the selection moves on.
  $effect(() => {
    const sel = current
    untrack(() => {
      diff = null
      diffError = ''
    })
    if (!sel) return
    const dir = repo
    const req =
      sel.kind === 'commit'
        ? { dir, commit: sel.commit.hash, file: '', untracked: false }
        : { dir, commit: '', file: sel.change.path, untracked: sel.change.index === '?' }
    backend
      .diff(win.serverID, req)
      .then((d) => {
        if (current === sel) diff = d
      })
      .catch((e: unknown) => {
        if (current === sel) diffError = toUserError(e).message
      })
  })

  // Load when the server comes up and when the window regains focus; no
  // polling, as every read is a command in the audit log.
  $effect(() => {
    if (connected) untrack(() => void findRepos())
  })
  let wasFocused = untrack(() => ctx.focused)
  $effect(() => {
    const f = ctx.focused
    if (f && !wasFocused && connected && repo) untrack(() => void loadStatus(repo))
    wasFocused = f
  })

  function setFolders(next: string[]) {
    folders = { ...folders, [win.serverID]: next }
    void findRepos()
  }
</script>

<FilterShell
  group={server?.name ?? ''}
  {filters}
  bind:filter={() => repo, (r) => (repo = r)}
  sidebarLabel={t.git.repos}
  searchLabel={t.git.search}
  bind:query
  refreshing={!connected || finding}
  ready={connected && (repos?.length ?? 0) > 0}
  {footer}
  {error}
  onrefresh={refresh}
>
  {#snippet actions()}
    <button
      class="tool"
      aria-label={t.git.addFolder}
      title={t.git.addFolder}
      disabled={!connected}
      onclick={() => (editingFolders = true)}
    >
      <Icon name="plus" size={15} strokeWidth={2.2} />
    </button>
  {/snippet}
  {#snippet placeholder()}
    {#if !connected}
      <Empty icon="server" title={t.git.offline} hint={t.git.offlineHint} />
    {:else if findError}
      <Empty icon="warning" title={findError} />
    {:else if repos === null}
      <Empty icon="gitBranch" title={t.git.finding} />
    {:else}
      <Empty icon="gitBranch" title={t.git.noRepos} hint={t.git.noReposHint} />
    {/if}
  {/snippet}
  {#snippet list()}
    <div class="tabs" role="group" aria-label={t.git.repos}>
      {#each ['changes', 'history'] as const as id (id)}
        <button
          type="button"
          class="tab"
          class:active={tab === id}
          aria-pressed={tab === id}
          onclick={() => ((tab = id), (selected = null))}
        >
          <Icon name={id === 'changes' ? 'file' : 'gitCommit'} size={13} />
          {t.git.tabs[id]}
          {#if id === 'changes' && status && status.changes.length > 0}
            <span class="count">{status.changes.length}</span>
          {/if}
        </button>
      {/each}
    </div>
    {#if tab === 'changes' && status?.truncated}<p class="warn">{t.git.truncated}</p>{/if}
    <StatusList
      items={shown}
      {selected}
      label={t.git.tabs[tab]}
      empty={query
        ? t.git.noMatch
        : status === null && !error
          ? t.git.loading
          : tab === 'changes'
            ? t.git.noChanges
            : commits === null
              ? t.git.loading
              : t.git.noCommits}
      onselect={(k) => (selected = k)}
    />
    {#if tab === 'history' && more && !query}
      <button type="button" class="more" onclick={() => void loadLog(repo, commits?.length ?? 0)}
        >{t.git.loadMore}</button
      >
    {/if}
  {/snippet}
  {#snippet detail()}
    <RepoInspector serverID={win.serverID} dir={repo} {status} sel={current} {diff} {diffError} />
  {/snippet}
</FilterShell>

<FoldersDialog
  open={editingFolders}
  folders={extra}
  onchange={setFolders}
  onclose={() => (editingFolders = false)}
/>

<style>
  .tabs {
    display: flex;
    gap: 4px;
    padding: 8px 12px 4px;
  }

  .tab {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    padding: 3px 10px;
    border: 0;
    border-radius: var(--control-radius);
    background: none;
    color: var(--fg-muted);
    font-size: var(--text-sm);
  }

  .tab:hover {
    background: var(--row-hover);
  }

  .tab.active {
    background: var(--fill-soft);
    color: var(--fg);
    font-weight: 600;
  }

  .count {
    padding: 0 6px;
    border-radius: 999px;
    background: var(--accent-soft);
    color: var(--accent);
    font-size: var(--text-xxs);
    font-variant-numeric: tabular-nums;
  }

  .warn {
    margin: 2px 16px;
    font-size: var(--text-xs);
    color: var(--warn);
  }

  .more {
    margin: 0 16px 8px;
    padding: 4px;
    border: 0;
    border-radius: var(--control-radius);
    background: none;
    color: var(--accent);
    font-size: var(--text-sm);
  }

  .more:hover {
    background: var(--row-hover);
  }
</style>
