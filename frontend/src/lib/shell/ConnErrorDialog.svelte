<script lang="ts">
  import { copyText } from '../api/clipboard'
  import { diagRevealLog, diagSupportInfo, type SupportInfo } from '../api/diag'
  import { toUserError } from '../api/errors'
  import { hasRuntime, openURL } from '../api/runtime'
  import { t } from '../i18n'
  import { servers } from '../stores/servers.svelte'
  import { ui } from '../stores/ui.svelte'
  import Dialog from '../ui/Dialog.svelte'
  import { connReport } from './connReport'

  const ISSUES_URL = 'https://github.com/2hjaito/davish/issues/new'

  const c = t.connError
  const srv = $derived(servers.list.find((s) => s.id === servers.failureOf))

  let info = $state.raw<SupportInfo | null>(null)
  let copied = $state(false)
  let note = $state('')
  const report = $derived(srv ? connReport(info, srv) : '')

  // Fresh version and log each time the dialog opens or the cause changes.
  $effect(() => {
    void srv?.errorDetail
    copied = false
    note = ''
    if (srv && hasRuntime) void diagSupportInfo().then((i) => (info = i))
  })

  async function copy() {
    copied = await copyText(report)
    if (!copied) note = c.copyFailed
  }

  async function reveal() {
    note = ''
    try {
      await diagRevealLog()
    } catch (e) {
      note = toUserError(e).message
    }
  }

  function edit() {
    if (!srv) return
    const id = srv.id
    servers.closeFailure()
    ui.editServer(id)
  }

  function reconnect() {
    if (!srv) return
    const id = srv.id
    servers.closeFailure()
    void servers.reconnect(id)
  }
</script>

<Dialog
  open={!!srv && srv.state === 'failed'}
  title={c.title(srv?.name ?? '')}
  onclose={() => servers.closeFailure()}
  width={520}
>
  {#if srv}
    <p class="lead" role="alert">{t.errors[srv.error ?? 'INTERNAL']}</p>
    <p class="hint">{c.hint}</p>
    <pre class="detail selectable">{report}</pre>
    <div class="actions">
      <button class="btn" onclick={() => void copy()}>{copied ? c.copied : c.copy}</button>
      {#if hasRuntime}
        <button class="btn" onclick={() => void reveal()}>{c.openLog}</button>
      {/if}
      <button class="btn link" onclick={() => openURL(ISSUES_URL)}>{c.report}</button>
    </div>
    {#if note}<p class="hint">{note}</p>{/if}
  {/if}

  {#snippet footer()}
    <button class="btn" onclick={edit}>{c.edit}</button>
    <span class="spacer"></span>
    <!-- svelte-ignore a11y_autofocus -->
    <button class="btn" autofocus onclick={() => servers.closeFailure()}>{c.close}</button>
    <button class="btn btn-primary" onclick={reconnect}>{c.reconnect}</button>
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
    color: var(--danger);
  }

  .hint {
    font-size: var(--text-xs);
    color: var(--fg-muted);
    line-height: 1.45;
  }

  .detail {
    max-height: 220px;
    overflow: auto;
    margin: 0 0 8px;
    padding: 8px 10px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    line-height: 1.5;
    white-space: pre-wrap;
    overflow-wrap: anywhere;
    user-select: text;
    cursor: text;
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

  .spacer {
    flex: 1;
  }
</style>
