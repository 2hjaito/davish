<script lang="ts" module>
  import type { Change, Commit } from '../../lib/api/git'

  /** A changed file or a commit picked in the list. */
  export type Selected = { kind: 'change'; change: Change } | { kind: 'commit'; commit: Commit }
</script>

<script lang="ts">
  import type { Diff, Status } from '../../lib/api/git'
  import { t } from '../../lib/i18n/vi'
  import Empty from '../../lib/ui/Empty.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import { openFilesAt, openTerminalAt } from '../../lib/wm/intents.svelte'
  import DiffView from './DiffView.svelte'
  import RepoActions from './RepoActions.svelte'

  interface Props {
    serverID: string
    dir: string
    status: Status | null
    sel: Selected | null
    diff: Diff | null
    diffError: string
    /** The repository may have changed: read it again. */
    onchanged: () => void
  }

  let { serverID, dir, status, sel, diff, diffError, onchanged }: Props = $props()

  const dateTime = (s: number) => new Date(s * 1000).toLocaleString('vi-VN')
</script>

{#if sel}
  <section class="pane" aria-label={sel.kind === 'commit' ? sel.commit.subject : sel.change.path}>
    <header>
      <span class="icon"
        ><Icon name={sel.kind === 'commit' ? 'gitCommit' : 'file'} size={16} /></span
      >
      <div class="heading">
        {#if sel.kind === 'commit'}
          <h3>{sel.commit.subject}</h3>
          <p class="sub">
            <code>{sel.commit.short}</code> · {sel.commit.author} · {dateTime(sel.commit.time)}
          </p>
        {:else}
          <h3 class="mono">{sel.change.path}</h3>
          {#if sel.change.origPath}
            <p class="sub">{t.git.renamedFrom(sel.change.origPath)}</p>
          {/if}
        {/if}
      </div>
    </header>
    {#if diffError}
      <p class="bad">{diffError}</p>
    {:else if diff}
      <DiffView text={diff.text} truncated={diff.truncated} />
    {/if}
  </section>
{:else if status}
  <section class="pane summary" aria-label={dir}>
    <dl class="facts">
      <dt>{t.git.path}</dt>
      <dd><code>{dir}</code></dd>
      <dt>{t.git.branch}</dt>
      <dd>
        {#if status.branch}<code>{status.branch}</code>{:else}{t.git.detached}{/if}
      </dd>
      <dt>{t.git.upstream}</dt>
      <dd>
        {#if status.upstream}
          <code>{status.upstream}</code>
          <span class="sync" class:behind={status.behind > 0} title={t.git.syncHint}>
            {t.git.sync(status.ahead, status.behind)}
          </span>
        {:else}
          {t.git.noUpstream}
        {/if}
      </dd>
      {#if status.last}
        <dt>{t.git.last}</dt>
        <dd>
          {status.last.subject}
          <span class="sub"
            ><code>{status.last.short}</code> · {status.last.author} · {dateTime(
              status.last.time,
            )}</span
          >
        </dd>
      {/if}
    </dl>
    <div class="actions">
      <button type="button" class="btn" onclick={() => openFilesAt(serverID, dir)}>
        <Icon name="files" size={13} />
        {t.git.openFiles}
      </button>
      <button type="button" class="btn" onclick={() => openTerminalAt(serverID, dir)}>
        <Icon name="terminal" size={13} />
        {t.git.openTerminal}
      </button>
    </div>
    <RepoActions {serverID} {dir} {status} {onchanged} />
  </section>
{:else}
  <Empty icon="gitBranch" title={t.git.loading} />
{/if}

<style>
  .pane {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }

  header {
    display: flex;
    gap: 10px;
    align-items: flex-start;
    padding: 14px 16px 10px;
    border-bottom: 1px solid var(--divider-soft);
  }

  .icon {
    display: flex;
    padding: 6px;
    border-radius: var(--item-radius);
    background: var(--accent-soft);
    color: var(--accent);
  }

  .heading {
    min-width: 0;
  }

  h3 {
    margin: 0;
    font-size: var(--text-md);
    overflow-wrap: anywhere;
  }

  h3.mono {
    font-family: var(--font-mono);
    font-size: var(--text-sm);
  }

  .sub {
    display: block;
    margin: 2px 0 0;
    color: var(--fg-muted);
    font-size: var(--text-xs);
  }

  .summary {
    gap: 12px;
    padding: 16px;
    overflow-y: auto;
  }

  .facts {
    display: grid;
    grid-template-columns: max-content 1fr;
    gap: 6px 14px;
    margin: 0;
    font-size: var(--text-sm);
  }

  dt {
    color: var(--fg-muted);
  }

  dd {
    margin: 0;
    min-width: 0;
    overflow-wrap: anywhere;
  }

  code {
    font-family: var(--font-mono);
    font-size: var(--text-xs);
  }

  .sync {
    display: block;
    color: var(--ok);
    font-size: var(--text-xs);
  }

  .sync.behind {
    color: var(--warn);
  }

  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .actions .btn {
    display: inline-flex;
    align-items: center;
    gap: 6px;
  }

  .bad {
    margin: 12px 16px;
    color: var(--danger);
    font-size: var(--text-sm);
  }
</style>
