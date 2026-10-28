<script lang="ts">
  import { t } from '../i18n/vi'
  import Dialog from './Dialog.svelte'

  interface Props {
    /** The dialog is closed while command is null. */
    command: string | null
    title: string
    /** What happens, ending with an introduction to the command. */
    body: string
    /** Label of the danger button. */
    action: string
    onclose: () => void
    onconfirm: () => void
  }

  let { command, title, body, action, onclose, onconfirm }: Props = $props()
</script>

<!-- A destructive action shows the exact command first (docs/SECURITY.md). -->
<Dialog open={command !== null} {title} width={480} {onclose}>
  <p>{body}</p>
  <pre class="command">{command}</pre>

  {#snippet footer()}
    <span class="spacer"></span>
    <!-- Cancelling is the safe choice, so it has the focus. -->
    <!-- svelte-ignore a11y_autofocus -->
    <button type="button" class="btn" autofocus onclick={onclose}>{t.common.cancel}</button>
    <button type="button" class="btn btn-danger" onclick={onconfirm}>{action}</button>
  {/snippet}
</Dialog>

<style>
  p {
    margin: 0 0 10px;
    line-height: 1.45;
  }

  .command {
    margin: 0;
    padding: 8px 10px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    font-family: var(--font-mono);
    font-size: var(--text-sm);
    white-space: pre-wrap;
    word-break: break-all;
    user-select: text;
  }

  .spacer {
    flex: 1;
  }
</style>
