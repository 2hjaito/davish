<script lang="ts">
  import { toUserError } from '../api/errors'
  import type { AuthMethod, ServerDraft } from '../api/servers'
  import { t } from '../i18n/vi'
  import { servers } from '../stores/servers.svelte'
  import { ui } from '../stores/ui.svelte'
  import Dialog from '../ui/Dialog.svelte'

  const COLORS = ['', '#2f7de1', '#16a067', '#e08a12', '#d6455a', '#7a52d6', '#6b7180']

  const editing = $derived(
    ui.serverDialog?.mode === 'edit'
      ? servers.list.find((s) => s.id === (ui.serverDialog as { id: string }).id)
      : undefined,
  )

  let draft = $state<ServerDraft>(blank())
  let error = $state('')
  let busy = $state(false)
  let confirmingDelete = $state(false)

  function blank(): ServerDraft {
    return {
      id: '',
      name: '',
      host: '',
      port: 22,
      user: '',
      auth: 'agent',
      keyPath: '~/.ssh/id_ed25519',
      color: '',
      password: '',
      passphrase: '',
    }
  }

  // Start from the server being edited, or a blank form, each time it opens.
  $effect(() => {
    if (!ui.serverDialog) return
    const s = editing
    draft = s
      ? {
          ...blank(),
          ...s,
          keyPath: s.keyPath ?? '~/.ssh/id_ed25519',
          color: s.color ?? '',
          password: '',
          passphrase: '',
        }
      : blank()
    error = ''
    confirmingDelete = false
  })

  const auths: { value: AuthMethod; label: string }[] = [
    { value: 'agent', label: t.serverDialog.authAgent },
    { value: 'key', label: t.serverDialog.authKey },
    { value: 'password', label: t.serverDialog.authPassword },
  ]

  async function submit(e: SubmitEvent) {
    e.preventDefault()
    busy = true
    error = ''
    try {
      const saved = await servers.save($state.snapshot(draft))
      ui.closeServerDialog()
      // Reconnect so edited credentials take effect at once.
      if (editing) await servers.reconnect(saved.id)
      servers.select(saved.id)
    } catch (err) {
      error = toUserError(err).message
    } finally {
      busy = false
    }
  }

  async function remove() {
    if (!editing) return
    busy = true
    try {
      await servers.remove(editing.id)
      ui.closeServerDialog()
    } catch (err) {
      error = toUserError(err).message
    } finally {
      busy = false
    }
  }
</script>

<Dialog
  open={ui.serverDialog !== null}
  title={editing ? t.serverDialog.editTitle : t.serverDialog.addTitle}
  onclose={() => ui.closeServerDialog()}
  width={440}
>
  <form id="server-form" class="form" onsubmit={submit}>
    <label class="field">
      <span>{t.serverDialog.name}</span>
      <input
        class="input"
        required
        maxlength="64"
        placeholder={t.serverDialog.namePlaceholder}
        bind:value={draft.name}
      />
    </label>

    <div class="row">
      <label class="field grow">
        <span>{t.serverDialog.host}</span>
        <input
          class="input"
          required
          placeholder={t.serverDialog.hostPlaceholder}
          autocapitalize="off"
          spellcheck="false"
          bind:value={draft.host}
        />
      </label>
      <label class="field port">
        <span>{t.serverDialog.port}</span>
        <input class="input" type="number" min="1" max="65535" required bind:value={draft.port} />
      </label>
    </div>

    <label class="field">
      <span>{t.serverDialog.user}</span>
      <input
        class="input"
        required
        autocapitalize="off"
        spellcheck="false"
        bind:value={draft.user}
      />
    </label>

    <div class="field">
      <span>{t.serverDialog.auth}</span>
      <div class="segmented" role="radiogroup" aria-label={t.serverDialog.auth}>
        {#each auths as a (a.value)}
          <button
            type="button"
            role="radio"
            aria-checked={draft.auth === a.value}
            onclick={() => (draft.auth = a.value)}>{a.label}</button
          >
        {/each}
      </div>
    </div>

    {#if draft.auth === 'agent'}
      <p class="hint">{t.serverDialog.authAgentHint}</p>
    {:else if draft.auth === 'key'}
      <label class="field">
        <span>{t.serverDialog.keyPath}</span>
        <input class="input mono" required spellcheck="false" bind:value={draft.keyPath} />
        <small>{t.serverDialog.keyPathHint}</small>
      </label>
      <label class="field">
        <span>{t.serverDialog.passphrase}</span>
        <input class="input" type="password" autocomplete="off" bind:value={draft.passphrase} />
        <small
          >{editing?.hasPassphrase ? t.serverDialog.secretSaved : t.serverDialog.secretHint}</small
        >
      </label>
    {:else if draft.auth === 'password'}
      <label class="field">
        <span>{t.serverDialog.password}</span>
        <input
          class="input"
          type="password"
          autocomplete="off"
          required={!editing?.hasPassword}
          bind:value={draft.password}
        />
        <small
          >{editing?.hasPassword ? t.serverDialog.secretSaved : t.serverDialog.secretHint}</small
        >
      </label>
    {/if}

    <div class="field">
      <span>{t.serverDialog.color}</span>
      <div class="swatches" role="radiogroup" aria-label={t.serverDialog.color}>
        {#each COLORS as c (c)}
          <button
            type="button"
            class="swatch"
            class:none={c === ''}
            role="radio"
            aria-checked={draft.color === c}
            aria-label={c || '—'}
            style:background={c || undefined}
            onclick={() => (draft.color = c)}
          ></button>
        {/each}
      </div>
    </div>

    {#if error}<p class="error" role="alert">{error}</p>{/if}

    {#if confirmingDelete && editing}
      <div class="confirm">
        <p>{t.serverDialog.confirmDelete(editing.name)}</p>
        <div class="confirm-actions">
          <button type="button" class="btn" onclick={() => (confirmingDelete = false)}
            >{t.serverDialog.cancel}</button
          >
          <button type="button" class="btn btn-danger" disabled={busy} onclick={remove}
            >{t.serverDialog.confirmDeleteButton}</button
          >
        </div>
      </div>
    {/if}
  </form>

  {#snippet footer()}
    {#if editing}
      <button type="button" class="btn delete" onclick={() => (confirmingDelete = true)}
        >{t.serverDialog.delete}</button
      >
    {/if}
    <span class="spacer"></span>
    <button type="button" class="btn" onclick={() => ui.closeServerDialog()}
      >{t.serverDialog.cancel}</button
    >
    <button type="submit" form="server-form" class="btn btn-primary" disabled={busy}
      >{t.serverDialog.save}</button
    >
  {/snippet}
</Dialog>

<style>
  .form {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .row {
    display: flex;
    gap: 10px;
  }

  .grow {
    flex: 1;
  }

  .port {
    width: 90px;
  }

  .segmented {
    display: flex;
    padding: 2px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
  }

  .segmented button {
    flex: 1;
    border: 0;
    border-radius: 5px;
    padding: 5px 8px;
    background: none;
    font-size: var(--text-sm);
    transition: background var(--dur-fast) var(--ease-out);
  }

  .segmented button[aria-checked='true'] {
    background: var(--content-bg);
    box-shadow: var(--switch-knob-shadow);
    font-weight: 600;
  }

  .swatches {
    display: flex;
    gap: 8px;
  }

  .swatch {
    width: 20px;
    height: 20px;
    padding: 0;
    border-radius: 50%;
    border: 2px solid var(--glass-border);
  }

  .swatch.none {
    background: var(--content-bg);
  }

  .swatch[aria-checked='true'] {
    border-color: var(--accent);
    box-shadow: 0 0 0 2px var(--accent-soft);
  }

  .hint {
    margin: -4px 0 0;
    font-size: var(--text-xs);
    color: var(--fg-muted);
  }

  .error {
    margin: 0;
    color: var(--danger);
    font-size: var(--text-sm);
  }

  .confirm {
    padding: 10px 12px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    font-size: var(--text-sm);
  }

  .confirm p {
    margin: 0 0 8px;
  }

  .confirm-actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
  }

  .delete {
    color: var(--danger);
    background: none;
    padding-inline: 4px;
  }

  .spacer {
    flex: 1;
  }
</style>
