<script lang="ts">
  import type { ActionPlan } from '../../lib/api/services'
  import { t } from '../../lib/i18n/vi'
  import Dialog from '../../lib/ui/Dialog.svelte'
  import { shortName } from './units'

  interface Props {
    /** Stop or restart waiting for the user; the dialog is closed while null. */
    plan: ActionPlan | null
    onclose: () => void
    onconfirm: (plan: ActionPlan) => void
  }

  let { plan, onclose, onconfirm }: Props = $props()

  // Only stop and restart are confirmed; start runs right away.
  const action = $derived(plan?.action === 'stop' ? 'stop' : 'restart')
</script>

<Dialog
  open={plan !== null}
  title={plan ? t.services.confirmTitle[action](shortName(plan.unit)) : ''}
  width={480}
  {onclose}
>
  {#if plan}
    <p>{t.services.confirmBody[action]}</p>
    <!-- The command comes from Go and is exactly what will run. -->
    <pre class="command">{plan.command}</pre>
  {/if}

  {#snippet footer()}
    <span class="spacer"></span>
    <!-- Cancelling is the safe choice, so it has the focus. -->
    <!-- svelte-ignore a11y_autofocus -->
    <button type="button" class="btn" autofocus onclick={onclose}>{t.services.cancel}</button>
    <button type="button" class="btn btn-danger" onclick={() => plan && onconfirm(plan)}>
      {t.services.actions[action]}
    </button>
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
