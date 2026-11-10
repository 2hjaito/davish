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
  <div class="cmd">
    <pre class="command">{command}</pre>
    {#if onterminal}
      <button
        type="button"
        class="term"
        aria-label={terminalLabel}
        title={terminalLabel}
        onclick={onterminal}
      >
        <Icon name="terminal" size={14} />
      </button>
    {/if}
  </div>
  {#if note}<p class="note">{note}</p>{/if}

  {#snippet footer()}
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

  .cmd {
    display: flex;
    align-items: flex-start;
    gap: 6px;
  }

  .command {
    flex: 1;
    min-width: 0;
    margin: 0;
    padding: 8px 10px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    font-family: var(--font-mono);
    font-size: var(--text-sm);
    white-space: pre-wrap;
    word-break: break-all;
    user-select: text;
    max-height: 40vh;
    overflow: auto;
  }

  .term {
    flex-shrink: 0;
    display: flex;
    padding: 7px;
    border: 0;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    color: var(--fg-muted);
  }

  .term:hover {
    color: var(--fg);
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
