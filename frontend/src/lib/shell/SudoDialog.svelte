<script lang="ts">
  import { toUserError } from '../api/errors'
  import { sudoSetPassword, sudoState } from '../api/sudo'
  import { t } from '../i18n'
  import { servers } from '../stores/servers.svelte'
  import { sudoPrompt } from '../stores/sudoPrompt.svelte'
  import Dialog from '../ui/Dialog.svelte'
  import Icon from '../ui/Icon.svelte'
  import { openTerminalAt } from '../wm/intents.svelte'

  const s = t.sudo
  const req = $derived(sudoPrompt.request)
  const server = $derived(servers.list.find((x) => x.id === req?.serverID))

  let password = $state('')
  let busy = $state(false)
  let error = $state('')
  // The user is in no group that usually may use sudo: warn first.
  let unlikely = $state(false)

  $effect(() => {
    if (req) {
      password = ''
      error = ''
      unlikely = false
      const id = req.serverID
      void sudoState(id).then(
        (st) => (unlikely = st === 'unlikely' && req?.serverID === id),
        () => {},
      )
    }
  })

  async function submit() {
    const r = req
    if (!r || !password) return
    busy = true
    error = ''
    try {
      await sudoSetPassword(r.serverID, password)
      password = ''
      r.done(true)
    } catch (e) {
      const ue = toUserError(e)
      error = ue.code === 'CMD_NEEDS_ROOT' ? s.noSudo : ue.message
    } finally {
      busy = false
    }
  }

  function toTerminal() {
    const r = req
    if (!r) return
    password = ''
    openTerminalAt(r.serverID, undefined, r.terminal)
    r.done(false)
  }

  function cancel() {
    password = ''
    req?.done(false)
  }
</script>

<Dialog open={req !== null} title={s.title(server?.name ?? '')} width={500} onclose={cancel}>
  {#if req}
    {#if unlikely}<p class="warn" role="status">
        <Icon name="warning" size={14} />
        {s.unlikely}
      </p>{/if}
    <p>{s.body}</p>
    <pre class="code">{req.command}</pre>
    <form
      onsubmit={(e) => {
        e.preventDefault()
        void submit()
      }}
    >
      <label class="field">
        <span>{s.password(server?.user ?? '')}</span>
        <!-- svelte-ignore a11y_autofocus -->
        <input class="input" type="password" bind:value={password} autocomplete="off" autofocus />
      </label>
    </form>
    <p class="note"><Icon name="lock" size={13} /> {s.kept}</p>
    {#if error}<p class="error" role="alert">{error}</p>{/if}
  {/if}
  {#snippet footer()}
    <button type="button" class="btn" class:btn-primary={unlikely} onclick={toTerminal}>
      <Icon name="terminal" size={13} />
      {s.terminal}
    </button>
    <span class="spacer"></span>
    <button type="button" class="btn" onclick={cancel}>{t.common.cancel}</button>
    <button
      type="button"
      class="btn"
      class:btn-primary={!unlikely}
      disabled={!password || busy}
      onclick={() => void submit()}>{busy ? s.checking : s.run}</button
    >
  {/snippet}
</Dialog>

<style>
  p {
    margin: 0 0 10px;
    line-height: 1.45;
  }

  .code {
    margin: 0 0 12px;
    padding: 8px 10px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    font-family: var(--font-mono);
    font-size: var(--text-sm);
    white-space: pre-wrap;
    word-break: break-all;
    user-select: text;
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: 4px;
    margin-bottom: 10px;
  }

  .note {
    display: flex;
    gap: 6px;
    font-size: var(--text-sm);
    color: var(--fg-muted);
  }

  .note :global(svg) {
    flex-shrink: 0;
    margin-top: 3px;
  }

  .error {
    color: var(--danger);
  }

  .warn {
    display: flex;
    gap: 6px;
    color: var(--warn);
  }

  .warn :global(svg) {
    flex-shrink: 0;
    margin-top: 3px;
  }

  .spacer {
    flex: 1;
  }
</style>
