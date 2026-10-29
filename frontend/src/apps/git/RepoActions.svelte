<script lang="ts">
  import { useAppMenu } from '../../lib/shell/menus.svelte'
  import { toUserError } from '../../lib/api/errors'
  import type { Branch, GitOp, OpResult, Status } from '../../lib/api/git'
  import { t } from '../../lib/i18n'
  import ConfirmCommand from '../../lib/ui/ConfirmCommand.svelte'
  import Dialog from '../../lib/ui/Dialog.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import { openSelected, openTerminalAt } from '../../lib/wm/intents.svelte'
  import { backend } from './backend'

  interface Props {
    serverID: string
    dir: string
    status: Status
    /** The repository may have changed: read it again. */
    onchanged: () => void
  }

  let { serverID, dir, status, onchanged }: Props = $props()

  /** A planned operation waiting for the user's yes. */
  let pending = $state.raw<{ op: GitOp; branch: string; command: string; body: string } | null>(
    null,
  )
  let busy = $state(false)
  let blocked = $state('')
  let error = $state('')
  let result = $state.raw<(OpResult & { op: GitOp }) | null>(null)
  let branches = $state.raw<Branch[] | null>(null)
  let picking = $state(false)

  // Another repository starts with nothing pending or shown.
  $effect(() => {
    void dir
    pending = null
    blocked = ''
    error = ''
    result = null
  })

  async function plan(op: GitOp, branch = '') {
    blocked = ''
    error = ''
    try {
      const p = await backend.plan(serverID, dir, op, branch)
      // A dirty tree never gets a pull to confirm (docs/PLAN.md M15).
      if (op === 'pull' && p.dirty > 0) {
        blocked = t.git.dirtyPull(p.dirty)
        return
      }
      const body = t.git.confirm[op].body
      pending = {
        op,
        branch,
        command: p.command,
        body: op === 'switch' && p.dirty > 0 ? `${t.git.dirtySwitch(p.dirty)} ${body}` : body,
      }
    } catch (e) {
      error = toUserError(e).message
    }
  }

  async function run() {
    const p = pending
    if (!p) return
    pending = null
    busy = true
    result = null
    try {
      result = { ...(await backend.run(serverID, dir, p.op, p.branch, p.command)), op: p.op }
    } catch (e) {
      const ue = toUserError(e)
      if (ue.code === 'GIT_DIRTY') blocked = ue.message
      else error = ue.message
    } finally {
      busy = false
      onchanged()
    }
  }

  async function pickBranch() {
    picking = true
    branches = null
    try {
      branches = await backend.branches(serverID, dir)
    } catch (e) {
      picking = false
      error = toUserError(e).message
    }
  }

  function resultText(r: OpResult & { op: GitOp }): string {
    if (!r.error) return t.git.done[r.op]
    return r.error in t.errors ? t.errors[r.error as keyof typeof t.errors] : t.errors.CMD_FAILED
  }

  useAppMenu(() => [
    { label: `${t.git.fetch}…`, disabled: busy, run: () => void plan('fetch') },
    { label: `${t.git.pull}…`, disabled: busy || !status.upstream, run: () => void plan('pull') },
  ])
</script>

<div class="actions">
  <button type="button" class="btn" disabled={busy} onclick={() => void plan('fetch')}>
    <Icon name="download" size={13} />
    {t.git.fetch}
  </button>
  <button
    type="button"
    class="btn"
    disabled={busy || !status.upstream}
    title={status.upstream ? '' : t.errors.GIT_NO_UPSTREAM}
    onclick={() => void plan('pull')}
  >
    <Icon name="arrowDown" size={13} />
    {t.git.pull}
  </button>
  <button type="button" class="btn" disabled={busy} onclick={() => void pickBranch()}>
    <Icon name="gitBranch" size={13} />
    {t.git.switchBranch}
  </button>
  {#if status.install}
    <button
      type="button"
      class="btn"
      title={t.git.installHint}
      onclick={() => openTerminalAt(serverID, dir, status.install)}
    >
      <Icon name="terminal" size={13} />
      {t.git.install(status.install)}
    </button>
  {/if}
</div>

{#if busy}<p class="muted">{t.git.running}</p>{/if}
{#if blocked}
  <p class="warn"><Icon name="warning" size={14} /> {blocked}</p>
{/if}
{#if error}<p class="bad">{error}</p>{/if}
{#if result}
  <div class="result" class:failed={result.error !== ''}>
    <p>{resultText(result)}</p>
    {#if result.error === 'GIT_AUTH_FAILED'}
      <p class="hint">{t.git.deployKeyHint}</p>
      <div class="actions">
        <button
          type="button"
          class="btn btn-primary"
          onclick={() =>
            openSelected('keys', serverID, 'server', {
              title: t.apps.keys,
              size: { w: 900, h: 580 },
            })}
        >
          <Icon name="key" size={13} />
          {t.git.deployKey}
        </button>
      </div>
    {/if}
    {#if result.output}
      <pre aria-label={t.git.output}>{result.output}</pre>
    {/if}
  </div>
{/if}

<ConfirmCommand
  command={pending?.command ?? null}
  title={pending ? t.git.confirm[pending.op].title : ''}
  body={pending?.body ?? ''}
  action={pending ? t.git.confirm[pending.op].action : ''}
  tone="primary"
  onclose={() => (pending = null)}
  onconfirm={() => void run()}
/>

<Dialog open={picking} title={t.git.branchesTitle} width={420} onclose={() => (picking = false)}>
  {#if branches === null}
    <p class="muted">{t.git.branchesLoading}</p>
  {:else}
    <ul class="branches">
      {#each branches as b (`${b.remote}:${b.name}`)}
        <li>
          <button
            type="button"
            class="branch"
            disabled={b.current}
            onclick={() => ((picking = false), void plan('switch', b.name))}
          >
            <Icon name="gitBranch" size={13} />
            <span class="name">{b.name}</span>
            {#if b.current}<span class="tag">{t.git.branchCurrent}</span>{/if}
            {#if b.remote}<span class="tag">{t.git.branchRemote}</span>{/if}
          </button>
        </li>
      {/each}
    </ul>
  {/if}
  {#snippet footer()}
    <span class="spacer"></span>
    <button type="button" class="btn" onclick={() => (picking = false)}>{t.common.cancel}</button>
  {/snippet}
</Dialog>

<style>
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

  .muted {
    margin: 0;
    color: var(--fg-muted);
    font-size: var(--text-xs);
  }

  .warn {
    display: flex;
    gap: 6px;
    align-items: flex-start;
    margin: 0;
    padding: 8px 10px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    font-size: var(--text-sm);
  }

  .warn :global(svg) {
    flex-shrink: 0;
    margin-top: 2px;
    color: var(--warn);
  }

  .bad {
    margin: 0;
    color: var(--danger);
    font-size: var(--text-sm);
  }

  .result {
    display: flex;
    flex-direction: column;
    gap: 8px;
    font-size: var(--text-sm);
  }

  .result p {
    margin: 0;
    color: var(--ok);
  }

  .result.failed p {
    color: var(--danger);
  }

  .result p.hint {
    color: var(--fg);
  }

  pre {
    margin: 0;
    padding: 8px 10px;
    max-height: 180px;
    overflow: auto;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    white-space: pre-wrap;
    word-break: break-word;
    user-select: text;
  }

  .branches {
    margin: 0;
    padding: 0;
    max-height: 320px;
    overflow-y: auto;
    list-style: none;
  }

  .branch {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    padding: 6px 8px;
    border: 0;
    border-radius: var(--item-radius);
    background: none;
    color: var(--fg);
    font-size: var(--text-sm);
    text-align: left;
  }

  .branch:hover:not(:disabled) {
    background: var(--row-hover);
  }

  .branch:disabled {
    color: var(--fg-muted);
  }

  .name {
    flex: 1;
    min-width: 0;
    overflow-wrap: anywhere;
    font-family: var(--font-mono);
    font-size: var(--text-xs);
  }

  .tag {
    padding: 0 6px;
    border-radius: 999px;
    background: var(--fill-soft);
    color: var(--fg-muted);
    font-size: var(--text-xxs);
  }

  .spacer {
    flex: 1;
  }
</style>
