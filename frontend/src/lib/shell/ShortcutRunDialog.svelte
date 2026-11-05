<script lang="ts">
  import type { ShortcutTarget } from '../api/shortcuts'
  import { t } from '../i18n'
  import Dialog from '../ui/Dialog.svelte'

  interface Props {
    /** The dialog is closed while target is null. */
    target: ShortcutTarget | null
    server: string
    /** The script is not the one allowed to run directly. */
    changed: boolean
    onclose: () => void
    onreview: () => void
    onrun: (remember: boolean) => void
  }

  let { target, server, changed, onclose, onreview, onrun }: Props = $props()
  const s = t.shortcuts

  let remember = $state(false)
  $effect(() => {
    if (target) remember = false
  })

  const title = $derived.by(() => {
    if (!target) return ''
    return changed ? s.changedTitle(target.name, server) : s.runTitle(target.name, server)
  })
</script>

<!-- Shows the exact command (docs/SECURITY.md); Cancel stays the safe default. -->
<Dialog open={target !== null} {title} width={500} {onclose}>
  <p class:warn={changed}>{changed ? s.changedBody : s.runBody}</p>
  <pre class="command">{target?.command}</pre>
  {#if target?.hash}
    <label class="remember">
      <input type="checkbox" bind:checked={remember} />
      {s.remember}
    </label>
  {:else}
    <p class="note">{s.noHash}</p>
  {/if}

  {#snippet footer()}
    <button type="button" class="btn" onclick={onreview}>{s.review}</button>
    <span class="spacer"></span>
    <!-- svelte-ignore a11y_autofocus -->
    <button type="button" class="btn" autofocus onclick={onclose}>{t.common.cancel}</button>
    <button
      type="button"
      class="btn {changed ? 'btn-danger' : 'btn-primary'}"
      onclick={() => onrun(remember)}
    >
      {changed ? s.runAnyway : s.run}
    </button>
  {/snippet}
</Dialog>

<style>
  p {
    margin: 0 0 10px;
    line-height: 1.45;
  }

  .warn {
    color: var(--danger);
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

  .remember {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-top: 12px;
  }

  .note {
    margin: 12px 0 0;
    color: var(--fg-muted);
  }

  .spacer {
    flex: 1;
  }
</style>
