<script lang="ts">
  import { toUserError } from '../../lib/api/errors'
  import { type Entry, filesPlanRemove, filesRemove, type RemovePlan } from '../../lib/api/files'
  import { t } from '../../lib/i18n/vi'
  import Dialog from '../../lib/ui/Dialog.svelte'
  import NeedsRoot from '../../lib/ui/NeedsRoot.svelte'

  interface Props {
    serverID: string
    /** What to delete; the dialog is closed while null. */
    entry: Entry | null
    onclose: () => void
    ondone: (e: Entry) => void
  }

  let { serverID, entry, onclose, ondone }: Props = $props()

  let plan = $state<RemovePlan | null>(null)
  let error = $state('')
  /** The server refused for lack of rights: the sudo form to offer. */
  let sudo = $state('')
  let typed = $state('')
  let busy = $state(false)

  // The command shown is the one Go will run, so it comes from Go.
  $effect(() => {
    const e = entry
    plan = null
    error = ''
    sudo = ''
    typed = ''
    if (e) void loadPlan(e)
  })

  async function loadPlan(e: Entry) {
    try {
      const p = await filesPlanRemove(serverID, e.path)
      if (entry?.path === e.path) plan = p
    } catch (err) {
      error = toUserError(err).message
    }
  }

  const explain = $derived(
    !plan
      ? ''
      : plan.kind === 'link'
        ? t.files.deleteLink
        : plan.nonEmpty
          ? t.files.deleteNonEmpty
          : plan.dir
            ? t.files.deleteDir
            : t.files.deleteFile,
  )
  // A folder with content is confirmed a second time by typing its name.
  const ready = $derived(!!plan && !busy && (!plan.nonEmpty || typed === plan.name))

  async function confirm() {
    if (!plan || !entry || !ready) return
    busy = true
    error = ''
    try {
      await filesRemove(serverID, plan)
      ondone(entry)
    } catch (err) {
      const ue = toUserError(err)
      // Without the rights, the way out is the same command with sudo.
      if (ue.code === 'FS_PERMISSION_DENIED') sudo = plan.sudo
      else error = ue.message
      // It changed since the user looked: show what would run now.
      if (ue.code === 'FS_STALE') void loadPlan(entry)
    } finally {
      busy = false
    }
  }
</script>

<Dialog
  open={entry !== null}
  title={entry ? t.files.deleteTitle(entry.name) : ''}
  width={480}
  {onclose}
>
  {#if !plan && !error}
    <p class="muted">{t.files.loadingPlan}</p>
  {/if}
  {#if plan}
    <p>{explain}</p>
    <pre class="command">{plan.command}</pre>
    {#if plan.nonEmpty}
      <label class="field confirm">
        <span>{t.files.typeName(plan.name)}</span>
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
  {/if}
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  {#if sudo}<NeedsRoot {serverID} command={sudo} onopen={onclose} />{/if}

  {#snippet footer()}
    <span class="spacer"></span>
    <!-- Cancelling is the safe choice, so it has the focus. -->
    <!-- svelte-ignore a11y_autofocus -->
    <button type="button" class="btn" autofocus onclick={onclose}>{t.files.cancel}</button>
    <button
      type="button"
      class="btn btn-danger"
      disabled={!ready || sudo !== ''}
      onclick={() => void confirm()}
    >
      {busy ? t.files.deleting : t.files.actions.delete}
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
    word-break: break-all;
    user-select: text;
  }

  .confirm {
    margin-bottom: 10px;
  }

  .spacer {
    flex: 1;
  }
</style>
