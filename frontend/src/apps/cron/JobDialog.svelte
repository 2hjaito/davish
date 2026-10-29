<script lang="ts">
  import { untrack } from 'svelte'

  import type { CronChange, CronJob, CronPreview } from '../../lib/api/cron'
  import { t } from '../../lib/i18n'
  import Dialog from '../../lib/ui/Dialog.svelte'
  import { backend } from './backend'
  import { describe } from './describe'
  import { formatRun } from './when'

  interface Props {
    /** Open for a new job ('new'), an existing one, or closed (null). */
    job: CronJob | 'new' | null
    zone: string
    busy: boolean
    /** Why the last plan failed. */
    error: string
    onclose: () => void
    onsubmit: (c: CronChange) => void
  }

  let { job, zone, busy, error, onclose, onsubmit }: Props = $props()

  let schedule = $state('')
  let command = $state('')
  let preview = $state.raw<CronPreview | null>(null)
  let bad = $state(false)
  const listId = $props.id()

  $effect(() => {
    const j = job
    untrack(() => {
      schedule = j && j !== 'new' ? j.schedule.raw : '0 3 * * *'
      command = j && j !== 'new' ? j.command : ''
    })
  })

  // Parsed by Go as the user types, a moment after they stop.
  $effect(() => {
    const raw = schedule
    if (!job) return
    const timer = setTimeout(() => {
      backend.preview(raw, zone).then(
        (p) => {
          if (raw !== schedule) return
          preview = p
          bad = false
        },
        () => {
          if (raw !== schedule) return
          preview = null
          bad = true
        },
      )
    }, 200)
    return () => clearTimeout(timer)
  })

  // An unescaped % ends the command in cron and feeds the rest to stdin.
  const percent = $derived(/(^|[^\\])%/.test(command))
  const ready = $derived(!bad && preview !== null && command.trim() !== '' && !busy)

  function submit(e: SubmitEvent) {
    e.preventDefault()
    if (!ready || !job) return
    onsubmit({
      op: job === 'new' ? 'add' : 'edit',
      id: job === 'new' ? '' : job.id,
      schedule: schedule.trim(),
      command: command.trim(),
    })
  }
</script>

<Dialog
  open={job !== null}
  title={job === 'new' ? t.cron.dialogAdd : t.cron.dialogEdit}
  width={520}
  {onclose}
>
  <form id="{listId}-form" onsubmit={submit}>
    <label class="field">
      <span>{t.cron.schedule}</span>
      <input
        class="input mono"
        type="text"
        bind:value={schedule}
        placeholder={t.cron.scheduleHint}
        spellcheck="false"
        autocomplete="off"
      />
    </label>
    <div class="presets">
      {#each t.cron.presets as [raw, label] (raw)}
        <button
          type="button"
          class="chip"
          aria-pressed={schedule.trim() === raw}
          onclick={() => (schedule = raw)}>{label}</button
        >
      {/each}
    </div>
    <p class="preview" class:bad aria-live="polite">
      {#if bad}{t.cron.invalidSchedule}
      {:else if preview}
        <strong>{describe(preview.schedule)}</strong>
        {#if preview.next[0]}· {t.cron.preview(formatRun(preview.next[0], zone))}{/if}
      {/if}
    </p>
    <label class="field">
      <span>{t.cron.command}</span>
      <input
        class="input mono"
        type="text"
        bind:value={command}
        placeholder={t.cron.commandHint}
        spellcheck="false"
        autocomplete="off"
      />
    </label>
    {#if percent}<p class="warn">{t.cron.percentWarning}</p>{/if}
    {#if error}<p class="warn" role="alert">{error}</p>{/if}
  </form>

  {#snippet footer()}
    <span class="spacer"></span>
    <button type="button" class="btn" onclick={onclose}>{t.common.cancel}</button>
    <button type="submit" form="{listId}-form" class="btn btn-primary" disabled={!ready}>
      {busy ? t.cron.planning : t.cron.continue}
    </button>
  {/snippet}
</Dialog>

<style>
  form {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .presets {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .chip {
    padding: 2px 10px;
    border: 0;
    border-radius: 11px;
    background: var(--fill-soft);
    font-size: var(--text-xs);
    color: var(--fg-muted);
  }

  .chip[aria-pressed='true'] {
    background: var(--accent-soft);
    color: var(--accent);
    font-weight: 600;
  }

  .preview {
    min-height: 18px;
    margin: 0 0 4px;
    font-size: var(--text-sm);
    color: var(--fg-muted);
  }

  .preview strong {
    color: var(--fg);
    font-weight: 600;
  }

  .preview.bad,
  .warn {
    color: var(--danger);
  }

  .warn {
    margin: 0;
    font-size: var(--text-xs);
    line-height: 1.4;
  }

  .spacer {
    flex: 1;
  }
</style>
