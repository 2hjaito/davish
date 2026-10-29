<script lang="ts">
  import { toUserError } from '../../lib/api/errors'
  import {
    type ArchiveFormat,
    type CompressPlan,
    type Entry,
    filesPlanCompress,
  } from '../../lib/api/files'
  import { t } from '../../lib/i18n'
  import Dialog from '../../lib/ui/Dialog.svelte'
  import { FORMATS } from './actions'

  interface Props {
    serverID: string
    /** What to compress; the dialog is closed while null. */
    entry: Entry | null
    onclose: () => void
    /** Runs the plan the user saw. */
    onconfirm: (e: Entry, format: ArchiveFormat, plan: CompressPlan) => void
  }

  let { serverID, entry, onclose, onconfirm }: Props = $props()

  let format = $state<ArchiveFormat>('zip')
  let plan = $state<CompressPlan | null>(null)
  let error = $state('')

  // Show the exact command for the chosen format before anything runs.
  $effect(() => {
    const e = entry
    const f = format
    plan = null
    error = ''
    if (!e) return
    filesPlanCompress(serverID, e.path, f)
      .then((p) => {
        if (entry?.path === e.path && format === f) plan = p
      })
      .catch((err: unknown) => (error = toUserError(err).message))
  })
</script>

<Dialog
  open={entry !== null}
  title={entry ? t.files.compressTitle(entry.name) : ''}
  width={480}
  {onclose}
>
  <div class="formats" role="radiogroup" aria-label={t.files.format}>
    {#each FORMATS as f (f.id)}
      <button
        type="button"
        role="radio"
        aria-checked={format === f.id}
        class:on={format === f.id}
        onclick={() => (format = f.id)}>{f.label}</button
      >
    {/each}
  </div>
  <p class="muted">{t.files.compressHint}</p>
  {#if plan}
    <pre class="command">{plan.command}</pre>
  {:else if !error}
    <p class="muted">{t.files.loadingPlan}</p>
  {/if}
  {#if error}<p class="error" role="alert">{error}</p>{/if}

  {#snippet footer()}
    <span class="spacer"></span>
    <button type="button" class="btn" onclick={onclose}>{t.files.cancel}</button>
    <button
      type="button"
      class="btn btn-primary"
      disabled={!plan}
      onclick={() => entry && plan && onconfirm(entry, format, plan)}
      >{t.files.actions.compress}</button
    >
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

  .error {
    color: var(--danger);
  }

  /* Segmented control, as the Monitor's range picker. */
  .formats {
    display: inline-flex;
    gap: 2px;
    margin-bottom: 10px;
    padding: 2px;
    border-radius: 8px;
    background: var(--fill-soft);
  }

  .formats button {
    padding: 3px 14px;
    border: 0;
    border-radius: 6px;
    background: none;
    font-weight: 600;
    color: var(--fg-muted);
  }

  .formats button.on {
    background: var(--chart-surface);
    color: var(--fg);
    box-shadow: var(--switch-knob-shadow);
  }

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
