<script lang="ts">
  import { copyText } from '../api/clipboard'
  import { diagRevealLog, diagSupportInfo, type SupportInfo } from '../api/diag'
  import { toUserError } from '../api/errors'
  import { hasRuntime, openURL } from '../api/runtime'
  import { t } from '../i18n'
  import { updates } from '../stores/updates.svelte'
  import Dialog from '../ui/Dialog.svelte'
  import { updateReport } from './updateReport'

  /** Where to download by hand when the check itself failed. */
  const RELEASES_URL = 'https://github.com/2hjaito/davish/releases/latest'

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

  // A failure says what went wrong up top, with a way forward and, on
  // request, what to send in a bug report.
  let showDetails = $state(false)
  let info = $state.raw<SupportInfo | null>(null)
  let copied = $state(false)
  let revealError = $state('')
  const report = $derived(updateReport(info, updates.errorDetail))

  $effect(() => {
    if (!updates.error) showDetails = false
  })

  function retry() {
    if (st?.available && latest?.signed) void updates.install()
    else void updates.check(true)
  }

  async function toggleDetails() {
    showDetails = !showDetails
    copied = false
    revealError = ''
    if (showDetails && hasRuntime) info = await diagSupportInfo()
  }

  async function copy() {
    copied = await copyText(report)
  }

  async function reveal() {
    revealError = ''
    try {
      await diagRevealLog()
    } catch (e) {
      revealError = toUserError(e).message
    }
  }
</script>

<Dialog open={updates.dialog} title={u.title} onclose={close} width={480}>
  {#if updates.error && !busy}
    <div class="failure" role="alert">
      <p class="lead">{u.failed}</p>
      <p>{updates.error}</p>
      <div class="actions">
        <button class="btn" disabled={updates.phase !== 'idle'} onclick={retry}>{u.retry}</button>
        <button class="btn" onclick={() => openURL(latest?.url ?? RELEASES_URL)}>{u.manual}</button>
        <button class="btn link" aria-expanded={showDetails} onclick={() => void toggleDetails()}
          >{showDetails ? u.hideDetails : u.details}</button
        >
      </div>
      {#if showDetails}
        <pre class="detail">{report}</pre>
        <p class="hint">{u.detailsHint}</p>
        <div class="actions">
          <button class="btn" onclick={() => void copy()}>{copied ? u.copied : u.copy}</button>
          {#if hasRuntime}
            <button class="btn" onclick={() => void reveal()}>{u.openLog}</button>
          {/if}
        </div>
        {#if revealError}<p class="hint">{revealError}</p>{/if}
      {/if}
    </div>
  {/if}
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

  .failure {
    margin: 0 0 12px;
    padding: 10px 12px;
    border-radius: var(--control-radius);
    border: 1px solid color-mix(in srgb, var(--danger) 45%, transparent);
    background: color-mix(in srgb, var(--danger) 10%, transparent);
  }

  .failure .lead {
    color: var(--danger);
  }

  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    margin-bottom: 4px;
  }

  .link {
    background: none;
    color: var(--accent);
  }

  .detail {
    max-height: 180px;
    overflow: auto;
    margin: 8px 0 6px;
    padding: 8px 10px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    white-space: pre-wrap;
    overflow-wrap: anywhere;
    user-select: text;
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
