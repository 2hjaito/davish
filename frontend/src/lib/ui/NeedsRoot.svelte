<script lang="ts">
  import { t } from '../i18n/vi'
  import { openTerminalAt } from '../wm/intents.svelte'
  import Icon from './Icon.svelte'

  interface Props {
    serverID: string
    /** The sudo command to prefill in a terminal. */
    command: string
    /** Why root is needed; a general sentence when not given. */
    message?: string
  }

  let { serverID, command, message = t.common.needsRoot }: Props = $props()
</script>

<!-- Davish never stores or types a sudo password (docs/SECURITY.md): the
     command is typed at a terminal prompt and the user presses Enter. -->
<div class="root" role="alert">
  <Icon name="lock" size={14} />
  <div class="body">
    <strong>{message}</strong>
    <p>{t.common.needsRootHint}</p>
    <pre class="command">{command}</pre>
    <button
      type="button"
      class="btn btn-primary"
      onclick={() => openTerminalAt(serverID, undefined, command)}
    >
      {t.common.runInTerminal}
    </button>
  </div>
</div>

<style>
  .root {
    display: flex;
    gap: 8px;
    padding: 10px 12px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    font-size: var(--text-sm);
  }

  .root > :global(svg) {
    flex-shrink: 0;
    margin-top: 2px;
    color: var(--warn);
  }

  .body {
    min-width: 0;
  }

  p {
    margin: 4px 0 8px;
    color: var(--fg-muted);
    line-height: 1.4;
  }

  .command {
    margin: 0 0 8px;
    padding: 6px 8px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    white-space: pre-wrap;
    word-break: break-all;
    user-select: text;
  }
</style>
