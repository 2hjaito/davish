<script lang="ts">
  import { toUserError } from '../../lib/api/errors'
  import type { CreateResult, EditPlan, NginxSite, SiteEdit } from '../../lib/api/nginx'
  import { t } from '../../lib/i18n'
  import Dialog from '../../lib/ui/Dialog.svelte'
  import NeedsRoot from '../../lib/ui/NeedsRoot.svelte'
  import { backend } from './backend'
  import { upstreamChoices, type Choice } from './upstreams'

  interface Props {
    serverID: string
    /** The site to change; the dialog is closed while null. */
    site: NginxSite | null
    onclose: () => void
    ondone: (name: string) => void
  }

  let { serverID, site, onclose, ondone }: Props = $props()

  let step = $state<'form' | 'preview' | 'result'>('form')
  let addNames = $state('')
  let upstream = $state('')
  let gzip = $state<SiteEdit['gzip']>('')
  let maxBody = $state('')
  let plan = $state.raw<EditPlan | null>(null)
  let result = $state.raw<CreateResult | null>(null)
  let error = $state('')
  let busy = $state(false)
  let suggestions = $state.raw<Choice[]>([])

  // The one address the site proxies to, if it has exactly one.
  const current = $derived.by(() => {
    const all = [...new Set(site?.servers.flatMap((s) => s.proxyPass) ?? [])]
    return all.length === 1 ? all[0] : ''
  })

  $effect(() => {
    if (!site) return
    step = 'form'
    addNames = ''
    upstream = ''
    gzip = ''
    maxBody = ''
    error = ''
    plan = null
    result = null
    if (current) void upstreamChoices(serverID).then((c) => (suggestions = c))
  })

  const edit = $derived<SiteEdit>({
    addNames: addNames.split(/[\s,]+/).filter(Boolean),
    upstream,
    gzip,
    maxBody,
  })

  async function preview() {
    if (!site) return
    busy = true
    error = ''
    try {
      plan = await backend.planEdit(serverID, site.path, edit)
      step = 'preview'
    } catch (e) {
      error = toUserError(e).message
    } finally {
      busy = false
    }
  }

  async function apply() {
    if (!plan || !site) return
    busy = true
    error = ''
    try {
      result = await backend.edit(serverID, edit, plan)
      if (result.created) {
        ondone(site.name)
        onclose()
      } else step = 'result'
    } catch (e) {
      error = toUserError(e).message
    } finally {
      busy = false
    }
  }
</script>

<Dialog
  open={site !== null}
  title={site ? t.nginx.edit.title(site.name) : ''}
  width={560}
  {onclose}
>
  {#if step === 'form'}
    <p class="muted">{t.nginx.edit.body}</p>
    <label class="field">
      <span>{t.nginx.edit.addNames}</span>
      <!-- svelte-ignore a11y_autofocus -->
      <input
        class="input"
        type="text"
        bind:value={addNames}
        placeholder="www.example.com"
        autofocus
        spellcheck="false"
      />
    </label>
    {#if current}
      <label class="field">
        <span>{t.nginx.edit.upstream}</span>
        <input
          class="input"
          type="text"
          list="edit-upstreams"
          bind:value={upstream}
          placeholder={current}
          spellcheck="false"
        />
        <datalist id="edit-upstreams">
          {#each suggestions as c (c.value)}<option value={c.value}>{c.label}</option>{/each}
        </datalist>
      </label>
    {/if}
    <div class="row">
      <label class="field">
        <span>{t.nginx.edit.gzip}</span>
        <select class="input" bind:value={gzip}>
          <option value="">{t.nginx.edit.keep}</option>
          <option value="on">{t.nginx.edit.on}</option>
          <option value="off">{t.nginx.edit.off}</option>
        </select>
      </label>
      <label class="field">
        <span>{t.nginx.edit.maxBody}</span>
        <input
          class="input"
          type="text"
          bind:value={maxBody}
          placeholder="50m"
          spellcheck="false"
        />
      </label>
    </div>
  {:else if step === 'preview' && plan}
    <p>{t.nginx.edit.previewBody(plan.backup)}</p>
    <pre class="config">{plan.after}</pre>
    <p>{t.nginx.commandIntro}</p>
    <pre class="command">{plan.command}</pre>
  {:else if step === 'result' && result}
    {#if result.needsRoot}
      <NeedsRoot
        {serverID}
        command={result.sudo}
        message={t.nginx.newSite.needsRoot}
        onopen={onclose}
      />
    {:else if result.test && !result.test.ok}
      <p class="error">{t.nginx.edit.testFailed}</p>
      <!-- Server output is shown as text only (docs/SECURITY.md). -->
      <pre class="command">{result.test.output}</pre>
    {/if}
  {/if}
  {#if error}<p class="error" role="alert">{error}</p>{/if}

  {#snippet footer()}
    {#if step !== 'form'}
      <button type="button" class="btn" onclick={() => (step = 'form')}
        >{t.nginx.newSite.back}</button
      >
    {/if}
    <span class="spacer"></span>
    <button type="button" class="btn" onclick={onclose}>{t.common.cancel}</button>
    {#if step === 'form'}
      <button type="button" class="btn btn-primary" disabled={busy} onclick={() => void preview()}
        >{t.nginx.newSite.next}</button
      >
    {:else if step === 'preview'}
      <button type="button" class="btn btn-primary" disabled={busy} onclick={() => void apply()}
        >{t.nginx.edit.apply}</button
      >
    {/if}
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

  .field {
    display: flex;
    flex: 1;
    flex-direction: column;
    gap: 4px;
    margin: 0 0 12px;
  }

  .row {
    display: flex;
    gap: 12px;
  }

  .config,
  .command {
    margin: 0 0 10px;
    padding: 8px 10px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    font-family: var(--font-mono);
    font-size: var(--text-sm);
    white-space: pre-wrap;
    word-break: break-all;
    user-select: text;
    overflow: auto;
  }

  .config {
    max-height: 260px;
  }

  .command {
    max-height: 120px;
  }

  .error {
    margin: 10px 0 0;
    color: var(--danger);
  }

  .spacer {
    flex: 1;
  }
</style>
