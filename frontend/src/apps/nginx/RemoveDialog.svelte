<script lang="ts">
  import { toUserError } from '../../lib/api/errors'
  import type { RemovePlan } from '../../lib/api/nginx'
  import { t } from '../../lib/i18n'
  import Dialog from '../../lib/ui/Dialog.svelte'

  interface Props {
    /** The dialog is closed while plan is null. */
    plan: RemovePlan | null
    /** Plans again with or without the certificates. */
    replan: (withCerts: boolean) => Promise<RemovePlan>
    onclose: () => void
    onconfirm: (plan: RemovePlan) => void
  }

  let { plan, replan, onclose, onconfirm }: Props = $props()

  // The plan shown follows the checkbox, so the command is always the one
  // that will run (docs/SECURITY.md).
  let shown = $state.raw<RemovePlan | null>(null)
  let error = $state('')
  $effect(() => {
    shown = plan
    error = ''
  })

  async function setCerts(withCerts: boolean) {
    try {
      shown = await replan(withCerts)
    } catch (e) {
      error = toUserError(e).message
    }
  }
</script>

<Dialog
  open={plan !== null}
  title={shown ? t.nginx.removeTitle(shown.name) : ''}
  width={500}
  {onclose}
>
  {#if shown}
    <p>{shown.enabled ? t.nginx.removeBodyEnabled : t.nginx.removeBody}</p>
    {#if shown.certs.length > 0}
      <label class="check">
        <input
          type="checkbox"
          checked={shown.withCerts}
          onchange={(e) => void setCerts(e.currentTarget.checked)}
        />
        <span>
          {t.nginx.removeCerts(shown.certs.join(', '))}
          <small>{t.nginx.removeCertsHint}</small>
        </span>
      </label>
    {/if}
    {#if shown.sharedCerts.length > 0}
      <p class="muted">{t.nginx.sharedCerts(shown.sharedCerts.join(', '))}</p>
    {/if}
    <p>{t.nginx.commandIntro}</p>
    <pre class="command">{shown.command}</pre>
    {#if error}<p class="error" role="alert">{error}</p>{/if}
  {/if}

  {#snippet footer()}
    <span class="spacer"></span>
    <!-- Cancelling is the safe choice, so it has the focus. -->
    <!-- svelte-ignore a11y_autofocus -->
    <button type="button" class="btn" autofocus onclick={onclose}>{t.common.cancel}</button>
    <button type="button" class="btn btn-danger" onclick={() => shown && onconfirm(shown)}
      >{t.nginx.removeAction}</button
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

  .check {
    display: flex;
    gap: 8px;
    align-items: flex-start;
    margin: 0 0 12px;
    padding: 8px 10px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
  }

  .check input {
    margin-top: 3px;
  }

  .check span {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  small {
    color: var(--fg-muted);
    font-size: var(--text-xs);
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

  .error {
    margin: 10px 0 0;
    color: var(--danger);
  }

  .spacer {
    flex: 1;
  }
</style>
