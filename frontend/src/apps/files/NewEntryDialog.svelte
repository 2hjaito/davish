<script lang="ts">
  import { toUserError } from '../../lib/api/errors'
  import { filesCreate, filesMkdir } from '../../lib/api/files'
  import { hasRuntime } from '../../lib/api/runtime'
  import { t } from '../../lib/i18n'
  import Dialog from '../../lib/ui/Dialog.svelte'
  import { renameProblem } from './actions'
  import { joinPath } from './listing'
  import { mockCreate } from './mockFs'

  interface Props {
    serverID: string
    /** The folder to create in. */
    dir: string | null
    /** What to create; the dialog is closed while null. */
    kind: 'file' | 'folder' | null
    onclose: () => void
    /** Called with the new path. */
    ondone: (path: string) => void
  }

  let { serverID, dir, kind, onclose, ondone }: Props = $props()

  let name = $state('')
  let error = $state('')
  let busy = $state(false)

  $effect(() => {
    if (!kind) return
    name = ''
    error = ''
  })

  const problem = $derived(renameProblem('', name))

  async function create(path: string, trimmed: string) {
    if (!hasRuntime) mockCreate(dir ?? '/', trimmed, kind === 'folder')
    else if (kind === 'folder') await filesMkdir(serverID, path)
    else await filesCreate(serverID, path)
  }

  async function submit() {
    if (!kind || dir === null || problem || busy) return
    busy = true
    error = ''
    const trimmed = name.trim()
    const path = joinPath(dir, trimmed)
    try {
      await create(path, trimmed)
      ondone(path)
    } catch (e) {
      error = toUserError(e).message
    } finally {
      busy = false
    }
  }
</script>

<Dialog
  open={kind !== null}
  title={kind === 'folder' ? t.files.newFolderTitle : t.files.newFileTitle}
  {onclose}
>
  <label class="field">
    <span>{t.files.entryName}</span>
    <!-- svelte-ignore a11y_autofocus -->
    <input
      class="input"
      type="text"
      bind:value={name}
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
      onclick={() => void submit()}>{t.files.create}</button
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
