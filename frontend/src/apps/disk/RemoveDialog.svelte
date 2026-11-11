<script lang="ts" module>
  /** Deleting a file or folder, or emptying a file such as a log. */
  export interface RemoveTarget {
    path: string
    size: number
    op: 'remove' | 'truncate'
  }
</script>

<script lang="ts">
  import { toUserError } from '../../lib/api/errors'
  import { withSudo } from '../../lib/stores/sudoPrompt.svelte'
  import { formatBytes } from '../../lib/format'
  import { t } from '../../lib/i18n'
  import Dialog from '../../lib/ui/Dialog.svelte'
  import NeedsRoot from '../../lib/ui/NeedsRoot.svelte'
  import { backend } from './backend'
  import { nameOf } from './tree'

  interface Props {
    serverID: string
    /** The dialog is closed while null. */
    target: RemoveTarget | null
    onclose: () => void
    ondone: (target: RemoveTarget) => void
  }

  let { serverID, target, onclose, ondone }: Props = $props()

  const d = t.disk

  /** Typing the name is asked from this size up, as for a folder with content. */
  const BIG = 1 << 30

  let command = $state('')
  let sudo = $state('')
  let nonEmpty = $state(false)
  let error = $state('')
  /** The server refused for lack of rights: offer the sudo form. */
  let needsRoot = $state(false)
  let typed = $state('')
  let busy = $state(false)

  const name = $derived(target ? nameOf(target.path) : '')
  const mustType = $derived(target?.op === 'remove' && (nonEmpty || (target?.size ?? 0) >= BIG))
  const ready = $derived(!!command && !busy && !needsRoot && (!mustType || typed === name))

  // The command shown is the one Go will run, so it comes from Go.
  $effect(() => {
    const tg = target
    command = ''
    sudo = ''
    nonEmpty = false
    error = ''
    needsRoot = false
    typed = ''
    if (tg) void plan(tg)
  })

  async function plan(tg: RemoveTarget) {
    try {
      if (tg.op === 'remove') {
        const p = await backend.planRemove(serverID, tg.path)
        if (target !== tg) return
        command = p.command
        sudo = p.sudo
        nonEmpty = p.nonEmpty
      } else {
        const p = await backend.planTruncate(tg.path)
        if (target !== tg) return
        command = p.command
        sudo = p.sudo
      }
    } catch (e) {
      const ue = toUserError(e)
      error = ue.code === 'INVALID_INPUT' ? d.protected : ue.message
    }
  }

  async function confirm() {
    const tg = target
    if (!tg || !ready) return
    busy = true
    error = ''
    try {
      if (tg.op === 'remove') {
        // As root through sudo when the server refuses (ADR 0018).
        const run = () => backend.remove(serverID, tg.path, command)
        if (!(await withSudo(serverID, sudo, run))) return
      } else await backend.truncate(serverID, tg.path, command)
      ondone(tg)
    } catch (e) {
      const ue = toUserError(e)
      // Without the rights, the way out is the same command with sudo.
      if (ue.code === 'FS_PERMISSION_DENIED') needsRoot = true
      else error = ue.message
      if (ue.code === 'FS_STALE') void plan(tg)
    } finally {
      busy = false
    }
  }
</script>

<Dialog
  open={target !== null}
  title={target ? (target.op === 'remove' ? d.removeTitle(name) : d.truncateTitle(name)) : ''}
  width={500}
  {onclose}
>
  {#if target && (command || !error)}
    <p>
      {target.op === 'truncate'
        ? d.truncateBody
        : nonEmpty
          ? d.removeDir(formatBytes(target.size))
          : d.removeFile(formatBytes(target.size))}
    </p>
  {/if}
  {#if command}
    <pre class="command">{command}</pre>
    {#if mustType}
      <label class="field confirm">
        <span>{t.files.typeName(name)}</span>
        <input
          class="input mono"
          type="text"
          bind:value={typed}
          autocomplete="off"
          spellcheck="false"
          onkeydown={(e) => e.key === 'Enter' && void confirm()}
        />
      </label>
    {/if}
  {:else if !error}
    <p class="muted">{t.files.loadingPlan}</p>
  {/if}
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  {#if needsRoot}<NeedsRoot {serverID} command={sudo} onopen={onclose} />{/if}

  {#snippet footer()}
    <span class="spacer"></span>
    <!-- Cancelling is the safe choice, so it has the focus. -->
    <!-- svelte-ignore a11y_autofocus -->
    <button type="button" class="btn" autofocus onclick={onclose}>{t.common.cancel}</button>
    <button type="button" class="btn btn-danger" disabled={!ready} onclick={() => void confirm()}>
      {busy ? d.working : target?.op === 'truncate' ? d.truncate : d.remove}
    </button>
  {/snippet}
</Dialog>

<style>
  p {
    margin: 0 0 10px;
    line-height: 1.45;
  }

  .muted {
    color: var(--fg-muted);
  }

  .error {
    color: var(--danger);
  }

  /* The exact command, as SECURITY.md asks of every destructive action. */
  .command {
    margin: 0 0 10px;
    padding: 8px 10px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    font-family: var(--font-mono);
    font-size: var(--text-sm);
    white-space: pre-wrap;
    overflow-wrap: anywhere;
    user-select: text;
  }

  .confirm {
    margin-bottom: 10px;
  }

  .spacer {
    flex: 1;
  }
</style>
