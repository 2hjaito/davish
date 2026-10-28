<script lang="ts">
  import type { CronChange, CronPlan } from '../../lib/api/cron'
  import { t } from '../../lib/i18n/vi'
  import Dialog from '../../lib/ui/Dialog.svelte'

  interface Props {
    /** The change waiting for the user; closed while null. */
    pending: { change: CronChange; plan: CronPlan } | null
    busy: boolean
    error: string
    onclose: () => void
    onconfirm: () => void
  }

  let { pending, busy, error, onclose, onconfirm }: Props = $props()

  const op = $derived(pending?.change.op ?? 'add')
</script>

<!-- Replacing the crontab shows the exact command and the lines it
     changes before anything runs (docs/SECURITY.md). -->
<Dialog open={pending !== null} title={t.cron.confirmTitle[op]} width={540} {onclose}>
  {#if pending}
    <div class="diff">
      {#if pending.plan.before}
        <div class="line removed">
          <span class="sign" aria-label={t.cron.removed}>−</span><code>{pending.plan.before}</code>
        </div>
      {/if}
      {#if pending.plan.after}
        <div class="line added">
          <span class="sign" aria-label={t.cron.added}>+</span><code>{pending.plan.after}</code>
        </div>
      {/if}
    </div>
    <p>{t.cron.confirmBody}</p>
    <pre class="command">{pending.plan.command}</pre>
    {#if pending.plan.backup}<p class="note">{t.cron.backupNote(pending.plan.backup)}</p>{/if}
    {#if error}<p class="error" role="alert">{error}</p>{/if}
  {/if}

  {#snippet footer()}
    <span class="spacer"></span>
    <!-- svelte-ignore a11y_autofocus -->
    <button type="button" class="btn" autofocus onclick={onclose}>{t.common.cancel}</button>
    <button
      type="button"
      class="btn {op === 'delete' ? 'btn-danger' : 'btn-primary'}"
      disabled={busy}
      onclick={onconfirm}
    >
      {busy ? t.cron.applying : t.cron.confirmAction[op]}
    </button>
  {/snippet}
</Dialog>

<style>
  p {
    margin: 10px 0 6px;
    line-height: 1.45;
  }

  .diff {
    display: flex;
    flex-direction: column;
    gap: 2px;
    border-radius: var(--control-radius);
    overflow: hidden;
  }

  .line {
    display: flex;
    gap: 8px;
    padding: 5px 10px;
    font-family: var(--font-mono);
    font-size: var(--text-sm);
  }

  .line code {
    white-space: pre-wrap;
    word-break: break-all;
    user-select: text;
  }

  .removed {
    background: var(--diff-removed);
  }

  .added {
    background: var(--diff-added);
  }

  .sign {
    font-weight: 700;
    color: var(--fg-muted);
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

  .note {
    font-size: var(--text-xs);
    color: var(--fg-muted);
  }

  .error {
    color: var(--danger);
  }

  .spacer {
    flex: 1;
  }
</style>
