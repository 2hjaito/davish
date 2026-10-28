<script lang="ts">
  import type { LocalKey } from '../../lib/api/keys'
  import { t } from '../../lib/i18n/vi'
  import Dialog from '../../lib/ui/Dialog.svelte'

  export type FormKind = 'authorized' | 'server' | 'local'

  /** What the form asks for, by kind. */
  export interface FormValue {
    text: string
    name: string
    comment: string
    passphrase: string
  }

  interface Props {
    /** Closed while null. */
    kind: FormKind | null
    busy: boolean
    error: string
    /** Local keys offered for pasting into authorized_keys. */
    local: LocalKey[]
    /** A comment to start with, such as user@server. */
    defaultComment: string
    onclose: () => void
    onsubmit: (v: FormValue) => void
  }

  let { kind, busy, error, local, defaultComment, onclose, onsubmit }: Props = $props()

  let text = $state('')
  let name = $state('')
  let comment = $state('')
  let passphrase = $state('')
  let again = $state('')

  // Each opening starts clean.
  $effect(() => {
    if (!kind) return
    text = ''
    name = kind === 'server' ? 'deploy_key' : 'davish_ed25519'
    comment = defaultComment
    passphrase = again = ''
  })

  const mismatch = $derived(kind === 'local' && passphrase !== again)
  const ready = $derived(
    kind === 'authorized' ? text.trim() !== '' : name.trim() !== '' && !mismatch,
  )

  function submit(e: SubmitEvent) {
    e.preventDefault()
    if (ready && !busy) onsubmit({ text: text.trim(), name: name.trim(), comment, passphrase })
  }
</script>

<Dialog open={kind !== null} title={kind ? t.keys.add[kind] : ''} width={500} {onclose}>
  <form id="key-form" onsubmit={submit}>
    {#if kind === 'authorized'}
      <label class="field">
        <span>{t.keys.paste}</span>
        <textarea class="input mono" rows="3" spellcheck="false" bind:value={text}></textarea>
        <small>{t.keys.pasteHint}</small>
      </label>
      {#if local.length > 0}
        <div class="picks" role="group" aria-label={t.keys.pickLocal}>
          {#each local as k (k.path)}
            <button
              type="button"
              class="btn"
              aria-pressed={text.trim() === k.public}
              onclick={() => (text = k.public)}>{k.name}</button
            >
          {/each}
        </div>
      {/if}
    {:else}
      <label class="field">
        <span>{t.keys.name}</span>
        <input class="input mono" spellcheck="false" autocomplete="off" bind:value={name} />
        <small>{t.keys.nameHint}</small>
      </label>
      <label class="field">
        <span>{t.keys.comment}</span>
        <input class="input" spellcheck="false" autocomplete="off" bind:value={comment} />
        <small>{t.keys.commentHint}</small>
      </label>
      {#if kind === 'local'}
        <label class="field">
          <span>{t.keys.passphrase}</span>
          <input
            class="input"
            type="password"
            autocomplete="new-password"
            bind:value={passphrase}
          />
          <small>{t.keys.passphraseHint}</small>
        </label>
        {#if passphrase}
          <label class="field">
            <span>{t.keys.passphraseAgain}</span>
            <input class="input" type="password" autocomplete="new-password" bind:value={again} />
            {#if again && mismatch}<small class="bad">{t.keys.mismatch}</small>{/if}
          </label>
        {/if}
      {/if}
    {/if}
    {#if error}<p class="bad" role="alert">{error}</p>{/if}
  </form>

  {#snippet footer()}
    <span class="spacer"></span>
    <button type="button" class="btn" onclick={onclose}>{t.keys.cancel}</button>
    <button type="submit" form="key-form" class="btn btn-primary" disabled={!ready || busy}
      >{kind === 'local' ? t.keys.generate : t.keys.next}</button
    >
  {/snippet}
</Dialog>

<style>
  form {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  textarea {
    resize: vertical;
    font-family: var(--font-mono);
    font-size: var(--text-xs);
  }

  .picks {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .picks .btn[aria-pressed='true'] {
    background: var(--accent-soft);
    color: var(--accent);
  }

  .bad {
    margin: 0;
    color: var(--danger);
  }

  .spacer {
    flex: 1;
  }
</style>
