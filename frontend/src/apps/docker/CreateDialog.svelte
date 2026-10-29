<script lang="ts">
  import { toUserError } from '../../lib/api/errors'
  import { t } from '../../lib/i18n'
  import Dialog from '../../lib/ui/Dialog.svelte'
  import { backend } from './backend'

  interface Props {
    serverID: string
    /** What to create; the dialog is closed while null. */
    kind: 'network' | 'volume' | null
    onclose: () => void
    oncreated: () => void
  }

  let { serverID, kind, onclose, oncreated }: Props = $props()
  const d = t.docker.create

  let name = $state('')
  let error = $state('')
  let busy = $state(false)

  $effect(() => {
    if (!kind) return
    name = ''
    error = ''
  })

  async function create() {
    if (!kind || !name.trim() || busy) return
    busy = true
    error = ''
    try {
      await backend.create(serverID, kind, name.trim())
      oncreated()
      onclose()
    } catch (e) {
      error = toUserError(e).message
    } finally {
      busy = false
    }
  }
</script>

<Dialog open={kind !== null} title={kind ? d.title[kind] : ''} {onclose}>
  <label class="field">
    <span>{d.name}</span>
    <!-- svelte-ignore a11y_autofocus -->
    <input
      class="input"
      type="text"
      bind:value={name}
      autofocus
      spellcheck="false"
      onkeydown={(e) => e.key === 'Enter' && void create()}
    />
  </label>
  {#if kind}<p class="cmd"><code>docker {kind} create {name.trim() || '…'}</code></p>{/if}
  {#if error}<p class="error" role="alert">{error}</p>{/if}

  {#snippet footer()}
    <span class="spacer"></span>
    <button type="button" class="btn" onclick={onclose}>{t.common.cancel}</button>
    <button
      type="button"
      class="btn btn-primary"
      disabled={busy || !name.trim()}
      onclick={() => void create()}>{d.action}</button
    >
  {/snippet}
</Dialog>

<style>
  .field {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .cmd {
    margin: 10px 0 0;
    color: var(--fg-muted);
    font-size: var(--text-sm);
  }

  .error {
    margin: 10px 0 0;
    color: var(--danger);
  }

  .spacer {
    flex: 1;
  }
</style>
