<script lang="ts">
  import { toUserError } from '../../lib/api/errors'
  import type { CreatePlan, CreateResult, NewSite, SiteOptions } from '../../lib/api/nginx'
  import { t } from '../../lib/i18n'
  import Dialog from '../../lib/ui/Dialog.svelte'
  import NeedsRoot from '../../lib/ui/NeedsRoot.svelte'
  import { backend } from './backend'

  interface Props {
    serverID: string
    open: boolean
    onclose: () => void
    /** After the site is written, enabled and nginx reloaded. */
    oncreated: (name: string, path: string) => void
  }

  let { serverID, open, onclose, oncreated }: Props = $props()

  type Step = 'form' | 'preview' | 'result'
  let step = $state<Step>('form')
  let names = $state('')
  let type = $state<NewSite['type']>('proxy')
  let root = $state('')
  let upstream = $state('127.0.0.1:3000')
  let websocket = $state(false)
  let phpSocket = $state('')
  let options = $state.raw<SiteOptions | null>(null)
  let config = $state('')
  let plan = $state.raw<CreatePlan | null>(null)
  let result = $state.raw<CreateResult | null>(null)
  let error = $state('')
  let busy = $state(false)

  $effect(() => {
    if (!open) return
    step = 'form'
    error = ''
    plan = null
    result = null
    backend
      .siteOptions(serverID)
      .then((o) => {
        options = o
        if (!phpSocket) phpSocket = o.phpSockets[0] ?? ''
      })
      .catch((e: unknown) => (error = toUserError(e).message))
  })

  const site = $derived<NewSite>({
    names: names.split(/[\s,]+/).filter(Boolean),
    type,
    root: root || `/var/www/${names.split(/[\s,]+/)[0] || 'site'}`,
    upstream,
    websocket,
    phpSocket,
  })

  async function preview() {
    busy = true
    error = ''
    try {
      config = await backend.renderSite(site)
      plan = await backend.planCreate(serverID, site, config)
      step = 'preview'
    } catch (e) {
      error = toUserError(e).message
    } finally {
      busy = false
    }
  }

  // The command follows the edited config, so what is shown is what runs.
  async function replan() {
    try {
      plan = await backend.planCreate(serverID, site, config)
      error = ''
    } catch (e) {
      error = toUserError(e).message
    }
  }

  async function create() {
    if (!plan) return
    busy = true
    error = ''
    try {
      result = await backend.create(serverID, site, plan)
      if (result.created) {
        oncreated(plan.name, plan.path)
        onclose()
      } else step = 'result'
    } catch (e) {
      error = toUserError(e).message
    } finally {
      busy = false
    }
  }
</script>

<Dialog {open} title={t.nginx.newSite.title} width={560} {onclose}>
  {#if step === 'form'}
    <label class="field">
      <span>{t.nginx.newSite.names}</span>
      <!-- svelte-ignore a11y_autofocus -->
      <input
        class="input"
        type="text"
        bind:value={names}
        placeholder="example.com www.example.com"
        autofocus
        autocomplete="off"
        spellcheck="false"
      />
    </label>
    <div class="types" role="radiogroup" aria-label={t.nginx.newSite.type}>
      {#each ['proxy', 'static', 'php'] as const as k (k)}
        <label class="type" class:on={type === k}>
          <input type="radio" name="type" value={k} bind:group={type} />
          <strong>{t.nginx.newSite.types[k]}</strong>
          <small>{t.nginx.newSite.typeHints[k]}</small>
        </label>
      {/each}
    </div>
    {#if type === 'proxy'}
      <label class="field">
        <span>{t.nginx.newSite.upstream}</span>
        <input class="input" type="text" bind:value={upstream} spellcheck="false" />
        <small>{t.nginx.newSite.upstreamHint}</small>
      </label>
      <label class="check">
        <input type="checkbox" bind:checked={websocket} />
        <span>{t.nginx.newSite.websocket}</span>
      </label>
    {:else}
      <label class="field">
        <span>{t.nginx.newSite.root}</span>
        <input
          class="input"
          type="text"
          bind:value={root}
          placeholder={site.root}
          spellcheck="false"
        />
        <small>{t.nginx.newSite.rootHint}</small>
      </label>
      {#if type === 'php'}
        <label class="field">
          <span>{t.nginx.newSite.phpSocket}</span>
          {#if options && options.phpSockets.length > 0}
            <select class="input" bind:value={phpSocket}>
              {#each options.phpSockets as s (s)}<option value={s}>{s}</option>{/each}
            </select>
          {:else}
            <input class="input" type="text" bind:value={phpSocket} spellcheck="false" />
            <small>{t.nginx.newSite.noPhp}</small>
          {/if}
        </label>
      {/if}
    {/if}
  {:else if step === 'preview' && plan}
    <p>{t.nginx.newSite.previewBody(plan.path)}</p>
    <textarea
      class="config"
      bind:value={config}
      onchange={() => void replan()}
      spellcheck="false"
      rows="14"></textarea>
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
      <p class="error">{t.nginx.newSite.testFailed}</p>
      <!-- Server output is shown as text only (docs/SECURITY.md). -->
      <pre class="command">{result.test.output}</pre>
    {/if}
  {/if}
  {#if error}<p class="error" role="alert">{error}</p>{/if}

  {#snippet footer()}
    {#if step === 'preview'}
      <button type="button" class="btn" onclick={() => (step = 'form')}
        >{t.nginx.newSite.back}</button
      >
    {/if}
    <span class="spacer"></span>
    <button type="button" class="btn" onclick={onclose}>{t.common.cancel}</button>
    {#if step === 'form'}
      <button
        type="button"
        class="btn btn-primary"
        disabled={busy || site.names.length === 0}
        onclick={() => void preview()}>{t.nginx.newSite.next}</button
      >
    {:else if step === 'preview'}
      <button
        type="button"
        class="btn btn-primary"
        disabled={busy || !!error}
        onclick={() => void create()}>{t.nginx.newSite.create}</button
      >
    {:else}
      <button type="button" class="btn" onclick={() => (step = 'preview')}
        >{t.nginx.newSite.back}</button
      >
    {/if}
  {/snippet}
</Dialog>

<style>
  p {
    margin: 0 0 10px;
    line-height: 1.45;
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: 4px;
    margin: 0 0 12px;
  }

  small {
    color: var(--fg-muted);
    font-size: var(--text-xs);
  }

  .types {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 8px;
    margin: 0 0 12px;
  }

  .type {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 8px 10px;
    border: 1px solid var(--border);
    border-radius: var(--control-radius);
    cursor: pointer;
  }

  .type.on {
    border-color: var(--accent);
    background: var(--fill-soft);
  }

  .type input {
    position: absolute;
    opacity: 0;
    pointer-events: none;
  }

  .check {
    display: flex;
    gap: 8px;
    align-items: center;
    margin: 0 0 12px;
  }

  .config,
  .command {
    box-sizing: border-box;
    width: 100%;
    margin: 0 0 10px;
    padding: 8px 10px;
    border: none;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    color: inherit;
    font-family: var(--font-mono);
    font-size: var(--text-sm);
  }

  .config {
    resize: vertical;
  }

  .command {
    max-height: 120px;
    overflow: auto;
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
