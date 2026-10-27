<script module lang="ts">
  export type Answer = 'replace' | 'keep' | 'skip'
</script>

<script lang="ts">
  import type { UploadItem } from '../../lib/api/transfer'
  import { t } from '../../lib/i18n/vi'
  import Dialog from '../../lib/ui/Dialog.svelte'

  interface Props {
    /** The clashing item; the dialog is closed while null. */
    item: UploadItem | null
    /** Clashes still waiting after this one. */
    remaining: number
    onanswer: (a: Answer) => void
  }

  let { item, remaining, onanswer }: Props = $props()
</script>

<Dialog
  open={item !== null}
  title={item ? t.files.conflictTitle(item.name) : ''}
  width={460}
  onclose={() => onanswer('skip')}
>
  {#if item}
    <p>{item.replaceCommand ? t.files.conflictFile : t.files.conflictDir}</p>
    {#if item.replaceCommand}
      <pre class="command">{item.replaceCommand}</pre>
    {/if}
    {#if remaining > 0}<p class="muted">{t.files.remaining(remaining)}</p>{/if}
  {/if}

  {#snippet footer()}
    <span class="spacer"></span>
    <!-- Skipping is the safe choice, so it has the focus. -->
    <!-- svelte-ignore a11y_autofocus -->
    <button type="button" class="btn" autofocus onclick={() => onanswer('skip')}
      >{t.files.skip}</button
    >
    <button type="button" class="btn" onclick={() => onanswer('keep')}>{t.files.keepBoth}</button>
    {#if item?.replaceCommand}
      <button type="button" class="btn btn-danger" onclick={() => onanswer('replace')}
        >{t.files.replace}</button
      >
    {/if}
  {/snippet}
</Dialog>

<style>
  p {
    margin: 0 0 10px;
    line-height: 1.45;
  }

  .muted {
    color: var(--fg-muted);
    font-size: var(--text-sm);
  }

  /* The exact operation, as SECURITY.md asks of every destructive action. */
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

  .spacer {
    flex: 1;
  }
</style>
