<script lang="ts">
  import { copyText } from '../../lib/api/clipboard'
  import {
    type AuthorizedKey,
    GIT_HOSTS,
    type GitHost,
    type GitTestResult,
    type LocalKey,
    type ServerKey,
  } from '../../lib/api/keys'
  import { t } from '../../lib/i18n/vi'
  import Icon from '../../lib/ui/Icon.svelte'

  /** The selected key, tagged with the section it is in. */
  export type Selected =
    | { kind: 'authorized'; key: AuthorizedKey }
    | { kind: 'server'; key: ServerKey }
    | { kind: 'local'; key: LocalKey }

  interface Props {
    sel: Selected
    busy: boolean
    /** A note after an action, good or bad. */
    flash: { text: string; ok: boolean } | null
    /** The last git test of this server key, if any. */
    git: { host: GitHost; result: GitTestResult } | null
    /** For a local key: on the server already, and whether the server logs in with it. */
    onServer: boolean
    inUse: boolean
    connected: boolean
    onremove: (k: AuthorizedKey) => void
    ongittest: (k: ServerKey, host: GitHost) => void
    onpush: (k: LocalKey) => void
    onuse: (k: LocalKey) => void
  }

  let {
    sel,
    busy,
    flash,
    git,
    onServer,
    inUse,
    connected,
    onremove,
    ongittest,
    onpush,
    onuse,
  }: Props = $props()

  let copied = $state(false)
  const title = $derived(sel.key.comment || ('name' in sel.key ? sel.key.name : t.keys.noComment))
  const publicLine = $derived(sel.kind === 'authorized' ? sel.key.text : sel.key.public)

  async function copy() {
    copied = await copyText(publicLine)
    setTimeout(() => (copied = false), 1500)
  }
</script>

<section class="inspector" aria-label={title}>
  <header>
    <span class="icon"><Icon name="key" size={18} /></span>
    <div class="heading">
      <h3>{title}</h3>
      {#if sel.kind !== 'authorized'}<p class="path"><code>{sel.key.path}</code></p>{/if}
    </div>
  </header>

  {#if sel.kind === 'authorized' && sel.key.login}
    <p class="badge">{t.keys.login}</p>
  {/if}

  <dl class="facts">
    <dt>{t.keys.type}</dt>
    <dd>
      {sel.key.type}
      {#if sel.kind === 'local'}· {sel.key.encrypted ? t.keys.encrypted : t.keys.unencrypted}{/if}
    </dd>
    <dt>{t.keys.fingerprint}</dt>
    <dd><code>{sel.key.fingerprint}</code></dd>
    {#if sel.kind === 'authorized' && sel.key.options.length > 0}
      <dt>{t.keys.options}</dt>
      <dd><code>{sel.key.options.join(',')}</code></dd>
    {/if}
  </dl>

  <div class="actions">
    <button type="button" class="btn" onclick={() => void copy()}>
      <Icon name="copy" size={13} />
      {copied ? t.keys.copied : t.keys.copy}
    </button>
    {#if sel.kind === 'authorized'}
      <button
        type="button"
        class="btn remove"
        disabled={busy}
        onclick={() => sel.kind === 'authorized' && onremove(sel.key)}>{t.keys.remove}</button
      >
    {:else if sel.kind === 'server'}
      {#each GIT_HOSTS as host (host)}
        <button
          type="button"
          class="btn"
          disabled={busy}
          onclick={() => sel.kind === 'server' && ongittest(sel.key, host)}
          >{t.keys.gitTest} {host}</button
        >
      {/each}
    {:else}
      <button
        type="button"
        class="btn"
        disabled={busy || !connected || onServer}
        onclick={() => sel.kind === 'local' && onpush(sel.key)}
        >{onServer ? t.keys.pushed : t.keys.pushToServer}</button
      >
      <button
        type="button"
        class="btn"
        disabled={busy || !onServer || inUse}
        title={onServer ? '' : t.keys.useHint}
        onclick={() => sel.kind === 'local' && onuse(sel.key)}
        >{inUse ? t.keys.inUse : t.keys.useForServer}</button
      >
    {/if}
  </div>

  {#if flash}
    <p class="flash" class:ok={flash.ok} role="status">{flash.text}</p>
  {/if}

  {#if git}
    <div class="git {git.result.outcome}" role="status">
      <strong>{t.keys.gitResult[git.result.outcome](git.host, git.result.account)}</strong>
      {#if git.result.output}<pre>{git.result.output}</pre>{/if}
    </div>
  {/if}

  <h4>{t.keys.publicKey}</h4>
  <!-- Public keys only; never a private one. Shown as text (docs/SECURITY.md). -->
  <pre class="public">{publicLine}</pre>
</section>

<style>
  .inspector {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 14px 16px 16px;
  }

  header {
    display: flex;
    gap: 10px;
    align-items: flex-start;
  }

  .icon {
    display: flex;
    padding: 6px;
    border-radius: var(--item-radius);
    background: var(--accent-soft);
    color: var(--accent);
  }

  .heading {
    min-width: 0;
  }

  h3 {
    margin: 0;
    font-size: var(--text-lg);
    overflow-wrap: anywhere;
  }

  .path {
    margin: 2px 0 0;
    color: var(--fg-muted);
    font-size: var(--text-xs);
    overflow-wrap: anywhere;
  }

  .badge {
    align-self: flex-start;
    margin: 0;
    padding: 2px 8px;
    border-radius: 999px;
    background: var(--accent-soft);
    color: var(--accent);
    font-size: var(--text-xs);
    font-weight: 600;
  }

  .facts {
    display: grid;
    grid-template-columns: max-content 1fr;
    gap: 4px 14px;
    margin: 0;
    font-size: var(--text-sm);
  }

  dt {
    color: var(--fg-muted);
  }

  dd {
    margin: 0;
    min-width: 0;
    overflow-wrap: anywhere;
  }

  code {
    font-family: var(--font-mono);
    font-size: var(--text-xs);
  }

  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .actions .btn {
    display: inline-flex;
    align-items: center;
    gap: 6px;
  }

  .remove {
    color: var(--danger);
  }

  .flash {
    margin: 0;
    font-size: var(--text-sm);
    color: var(--danger);
  }

  .flash.ok {
    color: var(--ok);
  }

  .git {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 10px 12px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    font-size: var(--text-sm);
  }

  .git.ok strong {
    color: var(--ok);
  }

  .git.denied strong,
  .git.hostkey strong,
  .git.unreachable strong {
    color: var(--danger);
  }

  .git pre {
    margin: 0;
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    white-space: pre-wrap;
    overflow-wrap: anywhere;
    color: var(--fg-muted);
  }

  h4 {
    margin: 6px 0 0;
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--fg-muted);
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .public {
    margin: 0;
    padding: 8px 10px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    white-space: pre-wrap;
    word-break: break-all;
    user-select: text;
  }
</style>
