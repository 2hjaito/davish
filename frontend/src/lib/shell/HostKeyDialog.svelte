<script lang="ts">
  import { toUserError } from '../api/errors'
  import { t } from '../i18n'
  import { servers } from '../stores/servers.svelte'
  import Dialog from '../ui/Dialog.svelte'

  const prompt = $derived(servers.hostKeyPrompt)
  let error = $state('')

  async function accept() {
    error = ''
    try {
      await servers.acceptHostKey()
    } catch (e) {
      error = toUserError(e).message
    }
  }
</script>

<!-- Trust on first use: the user must see the exact fingerprint before
     anything is pinned. A changed key never reaches this dialog. -->
<Dialog
  open={prompt !== null || error !== ''}
  title={t.hostKey.title}
  onclose={() => {
    error = ''
    servers.rejectHostKey()
  }}
  width={460}
>
  {#if prompt}
    <p class="body">{t.hostKey.body(prompt.host)}</p>
    <dl>
      <dt>{t.hostKey.keyType}</dt>
      <dd class="mono">{prompt.keyType}</dd>
      <dt>{t.hostKey.fingerprint}</dt>
      <dd class="mono selectable">{prompt.fingerprint}</dd>
    </dl>
  {/if}
  {#if error}<p class="error" role="alert">{error}</p>{/if}

  {#snippet footer()}
    <!-- Cancel comes first and takes focus, like other confirmations. -->
    <!-- svelte-ignore a11y_autofocus -->
    <button
      class="btn"
      autofocus
      onclick={() => {
        error = ''
        servers.rejectHostKey()
      }}>{t.hostKey.reject}</button
    >
    {#if prompt}
      <button class="btn btn-primary" onclick={accept}>{t.hostKey.accept}</button>
    {/if}
  {/snippet}
</Dialog>

<style>
  .body {
    margin: 0 0 12px;
    font-size: var(--text-sm);
    color: var(--fg-muted);
    line-height: 1.5;
  }

  dl {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: 6px 12px;
    margin: 0;
    padding: 10px 12px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
  }

  dt {
    font-size: var(--text-xs);
    color: var(--fg-muted);
    align-self: center;
  }

  dd {
    margin: 0;
    word-break: break-all;
  }

  .error {
    margin: 10px 0 0;
    color: var(--danger);
    font-size: var(--text-sm);
  }
</style>
