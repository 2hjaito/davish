<script lang="ts">
  import { openURL } from '../api/runtime'
  import { t } from '../i18n/vi'
  import { updates } from '../stores/updates.svelte'
  import Dialog from '../ui/Dialog.svelte'

  const u = t.update
  const st = $derived(updates.status)
  const latest = $derived(st?.latest)
  const busy = $derived(updates.phase === 'downloading')

  // Release notes are Markdown; headings and bullets read fine as text
  // once their markers are gone.
  const notes = $derived(
    (latest?.notes ?? '')
      .split('\n')
      .map((l) =>
        l
          .replace(/^#+\s*/, '')
          .replace(/^[-*]\s+/, '• ')
          .replace(/\*\*|`/g, ''),
      )
      .join('\n')
      .trim(),
  )

  const close = () => {
    if (!busy) updates.dialog = false
  }
</script>

<Dialog open={updates.dialog} title={u.title} onclose={close} width={480}>
  {#if updates.phase === 'checking'}
    <p>{u.checking}</p>
  {:else if updates.phase === 'ready'}
    <p>{u.ready(latest?.version ?? '')}</p>
  {:else if st?.available && latest}
    <p class="lead">{u.newer(latest.version, st.current)}</p>
    {#if notes}<pre class="notes">{notes}</pre>{/if}
    {#if !latest.signed}<p class="hint">{u.unsigned}</p>{/if}
    {#if busy}
      <div class="bar" role="progressbar" aria-valuenow={Math.round(updates.progress * 100)}>
        <span style:width="{updates.progress * 100}%"></span>
      </div>
      <p class="hint">{u.downloading(Math.round(updates.progress * 100))}</p>
    {/if}
  {:else if st}
    <p>{u.upToDate(st.current)}</p>
  {/if}
  {#if updates.error}<p class="error" role="alert">{updates.error}</p>{/if}

  {#snippet footer()}
    {#if updates.phase === 'ready'}
      <span class="spacer"></span>
      <button class="btn" onclick={close}>{u.later}</button>
      <button class="btn btn-primary" onclick={() => void updates.restart()}>{u.restart}</button>
    {:else if st?.available && latest}
      <button class="btn skip" disabled={busy} onclick={() => updates.skip()}>{u.skip}</button>
      <span class="spacer"></span>
      <button class="btn" disabled={busy} onclick={close}>{u.later}</button>
      {#if latest.signed}
        <button class="btn btn-primary" disabled={busy} onclick={() => void updates.install()}
          >{busy ? u.installing : u.install}</button
        >
      {:else}
        <button class="btn btn-primary" onclick={() => openURL(latest.url)}>{u.openPage}</button>
      {/if}
    {:else}
      <span class="spacer"></span>
      <button class="btn btn-primary" onclick={close}>{u.ok}</button>
    {/if}
  {/snippet}
</Dialog>

<style>
  p {
    margin: 0 0 8px;
    font-size: var(--text-sm);
  }

  .lead {
    font-size: var(--text-md);
    font-weight: 600;
  }

  .notes {
    max-height: 260px;
    overflow: auto;
    margin: 0 0 10px;
    padding: 10px 12px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    font-family: inherit;
    font-size: var(--text-sm);
    line-height: 1.45;
    white-space: pre-wrap;
  }

  .hint {
    font-size: var(--text-xs);
    color: var(--fg-muted);
  }

  .bar {
    height: 6px;
    border-radius: 3px;
    background: var(--fill-track);
    overflow: hidden;
    margin-bottom: 4px;
  }

  .bar span {
    display: block;
    height: 100%;
    background: var(--accent);
    transition: width var(--dur-fast) linear;
  }

  .error {
    color: var(--danger);
  }

  .skip {
    background: none;
    padding-inline: 4px;
    color: var(--fg-muted);
  }

  .spacer {
    flex: 1;
  }
</style>
