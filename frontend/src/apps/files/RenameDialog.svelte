<script lang="ts">
  import { toUserError } from '../../lib/api/errors'
  import { type Entry, filesRename } from '../../lib/api/files'
  import { t } from '../../lib/i18n'
  import Dialog from '../../lib/ui/Dialog.svelte'
  import { renameProblem, stemLength } from './actions'
  import { joinPath, parentPath } from './listing'

  interface Props {
    serverID: string
    /** What to rename; the dialog is closed while null. */
    entry: Entry | null
    onclose: () => void
    /** Called with the new path. */
    ondone: (path: string) => void
  }

  let { serverID, entry, onclose, ondone }: Props = $props()

  let name = $state('')
  let error = $state('')
  let busy = $state(false)
  let input = $state<HTMLInputElement>()

  $effect(() => {
    if (!entry) return
    name = entry.name
    error = ''
  })

  // Like Finder: the name is selected without its extension.
  $effect(() => {
    if (input && entry) input.setSelectionRange(0, stemLength(entry.name))
  })

  const problem = $derived(entry ? renameProblem(entry.name, name) : 'empty')

  async function submit() {
    if (!entry || problem || busy) return
    busy = true
    error = ''
    const to = joinPath(parentPath(entry.path), name.trim())
    try {
      await filesRename(serverID, entry.path, to)
      ondone(to)
    } catch (e) {
      error = toUserError(e).message
    } finally {
      busy = false
    }
  }
</script>

<Dialog open={entry !== null} title={entry ? t.files.renameTitle(entry.name) : ''} {onclose}>
  <label class="field">
    <span>{t.files.newName}</span>
    <!-- svelte-ignore a11y_autofocus -->
    <input
      class="input"
      type="text"
      bind:value={name}
      bind:this={input}
      autofocus
      autocomplete="off"
      spellcheck="false"
      onkeydown={(e) => e.key === 'Enter' && void submit()}
    />
  </label>
  {#if problem === 'slash'}<p class="error">{t.files.renameSlash}</p>{/if}
  {#if error}<p class="error" role="alert">{error}</p>{/if}

  {#snippet footer()}
    <span class="spacer"></span>
    <button type="button" class="btn" onclick={onclose}>{t.files.cancel}</button>
    <button
      type="button"
      class="btn btn-primary"
      disabled={!!problem || busy}
      onclick={() => void submit()}>{t.files.actions.rename}</button
    >
  {/snippet}
</Dialog>

<style>
  .error {
    margin: 8px 0 0;
    color: var(--danger);
    font-size: var(--text-sm);
  }

  .spacer {
    flex: 1;
  }
</style>
