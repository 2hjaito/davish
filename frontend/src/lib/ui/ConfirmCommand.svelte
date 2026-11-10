<script lang="ts">
  import { t } from '../i18n'
  import Dialog from './Dialog.svelte'
  import Icon from './Icon.svelte'

  interface Props {
    /** The dialog is closed while command is null. */
    command: string | null
    title: string
    /** What happens, ending with an introduction to the command. */
    body: string
    /** Label of the action button. */
    action: string
    /** Danger for what interrupts or removes something, primary otherwise. */
    tone?: 'danger' | 'primary'
    /** A note under the command, such as a warning. */
    note?: string
    /** Set to offer a small button that types the command into Terminal instead. */
    onterminal?: () => void
    terminalLabel?: string
    /** The action runs: its button waits. */
    busy?: boolean
    onclose: () => void
    onconfirm: () => void
  }

  let {
    command,
    title,
    body,
    action,
    tone = 'danger',
    note = '',
    onterminal,
    terminalLabel = t.common.runInTerminal,
    busy = false,
    onclose,
    onconfirm,
  }: Props = $props()
</script>

<!-- A destructive action shows the exact command first (docs/SECURITY.md). -->
<Dialog open={command !== null} {title} width={480} {onclose}>
  <p>{body}</p>
  <pre class="command">{command}</pre>
  {#if note}<p class="note">{note}</p>{/if}

  {#snippet footer()}
    {#if onterminal}
      <button type="button" class="btn" disabled={busy} onclick={onterminal}>
        <Icon name="terminal" size={13} />
        {terminalLabel}
      </button>
    {/if}
    <span class="spacer"></span>
    <!-- Cancelling is the safe choice, so it has the focus. -->
    <!-- svelte-ignore a11y_autofocus -->
    <button type="button" class="btn" autofocus onclick={onclose}>{t.common.cancel}</button>
    <button type="button" class="btn btn-{tone}" disabled={busy} onclick={onconfirm}
      >{action}</button
    >
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
    /* Wrap at spaces, so an address or a path is cut only when it must. */
    overflow-wrap: anywhere;
    user-select: text;
    max-height: 40vh;
    overflow: auto;
  }

  .note {
    margin: 10px 0 0;
    font-size: var(--text-sm);
    color: var(--warn);
  }

  .spacer {
    flex: 1;
  }
</style>
