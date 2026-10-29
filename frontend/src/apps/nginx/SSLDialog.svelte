<script lang="ts">
  import { toUserError } from '../../lib/api/errors'
  import type { SSLPlan } from '../../lib/api/nginx'
  import { t } from '../../lib/i18n'
  import Dialog from '../../lib/ui/Dialog.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import { openTerminalAt } from '../../lib/wm/intents.svelte'
  import { backend } from './backend'

  interface Props {
    serverID: string
    /** The site's domains; the dialog is closed while null. */
    names: string[] | null
    onclose: () => void
  }

  let { serverID, names, onclose }: Props = $props()

  let redirect = $state(true)
  let plan = $state.raw<SSLPlan | null>(null)
  let error = $state('')

  $effect(() => {
    if (!names) return
    void load(names, redirect)
  })

  async function load(n: string[], r: boolean) {
    error = ''
    try {
      plan = await backend.planSSL(serverID, n, r)
    } catch (e) {
      error = toUserError(e).message
    }
  }

  const unresolved = $derived(plan?.dns.filter((d) => !d.ok) ?? [])

  function open() {
    if (!plan?.sudo) return
    openTerminalAt(serverID, undefined, plan.sudo)
    onclose()
  }
</script>

<Dialog open={names !== null} title={t.nginx.ssl.title} width={520} {onclose}>
  {#if plan}
    {#if plan.names.length === 0}
      <p>{t.nginx.ssl.noNames}</p>
    {:else}
      <p>{t.nginx.ssl.body}</p>
      <ul class="dns">
        {#each plan.dns as d (d.name)}
          <li class:ok={d.ok}>
            <Icon name={d.ok ? 'check' : 'warning'} size={14} />
            <code>{d.name}</code>
            <span class="muted">{d.error ? t.nginx.ssl.noRecord : d.ips.join(', ')}</span>
          </li>
        {/each}
      </ul>
      {#if unresolved.length > 0}
        <p class="warn">
          {t.nginx.ssl.dnsWarn(plan.serverIPs.join(', ') || '?')}
        </p>
      {/if}
      {#if plan.skipped.length > 0}
        <p class="muted">{t.nginx.ssl.skipped(plan.skipped.join(', '))}</p>
      {/if}
      <label class="check">
        <input type="checkbox" bind:checked={redirect} />
        <span>{t.nginx.ssl.redirect}</span>
      </label>
      <p>{t.nginx.ssl.commandIntro}</p>
      <pre class="command">{plan.sudo}</pre>
    {/if}
  {/if}
  {#if error}<p class="warn" role="alert">{error}</p>{/if}

  {#snippet footer()}
    <span class="spacer"></span>
    <button type="button" class="btn" onclick={onclose}>{t.common.cancel}</button>
    <button type="button" class="btn btn-primary" disabled={!plan?.sudo} onclick={open}
      >{t.nginx.ssl.run}</button
    >
  {/snippet}
</Dialog>

<style>
  p {
    margin: 0 0 10px;
    line-height: 1.45;
  }

  .dns {
    margin: 0 0 10px;
    padding: 0;
    list-style: none;
  }

  .dns li {
    display: flex;
    gap: 8px;
    align-items: center;
    padding: 3px 0;
    color: var(--warn);
  }

  .dns li.ok {
    color: var(--ok);
  }

  .dns code {
    color: var(--fg);
  }

  .muted {
    color: var(--fg-muted);
    font-size: var(--text-sm);
  }

  .warn {
    color: var(--warn);
  }

  .check {
    display: flex;
    gap: 8px;
    align-items: center;
    margin: 0 0 12px;
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
