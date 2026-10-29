<script lang="ts">
  import type { StackPlan, StackTemplate } from '../../lib/api/docker'
  import { toUserError } from '../../lib/api/errors'
  import { t } from '../../lib/i18n'
  import Dialog from '../../lib/ui/Dialog.svelte'
  import { openFileInEditor, openTerminalAt } from '../../lib/wm/intents.svelte'
  import { backend } from './backend'

  interface Props {
    serverID: string
    open: boolean
    onclose: () => void
  }

  let { serverID, open, onclose }: Props = $props()
  const d = t.docker.stack

  let step = $state<'form' | 'preview' | 'done'>('form')
  let dir = $state('/srv/app')
  let template = $state<StackTemplate>('blank')
  let plan = $state.raw<StackPlan | null>(null)
  let error = $state('')
  let busy = $state(false)

  $effect(() => {
    if (!open) return
    step = 'form'
    error = ''
    plan = null
  })

  async function preview() {
    error = ''
    try {
      plan = await backend.planStack(dir, template)
      step = 'preview'
    } catch (e) {
      error = toUserError(e).message
    }
  }

  async function create() {
    if (!plan) return
    busy = true
    error = ''
    try {
      await backend.createStack(serverID, dir, template, plan.command)
      step = 'done'
    } catch (e) {
      error = toUserError(e).message
    } finally {
      busy = false
    }
  }

  function edit() {
    if (plan) openFileInEditor(serverID, plan.path)
    onclose()
  }

  // compose names a project after its folder, lowercased.
  async function up() {
    if (!plan) return
    const folder = plan.path.replace(/\/compose\.yaml$/, '')
    const project = (folder.split('/').pop() ?? '').toLowerCase().replace(/[^a-z0-9_-]/g, '')
    try {
      const cmd = await backend.composeCommand(project, folder, plan.path, 'up')
      openTerminalAt(serverID, folder, cmd)
      onclose()
    } catch (e) {
      error = toUserError(e).message
    }
  }
</script>

<Dialog {open} title={d.title} width={540} {onclose}>
  {#if step === 'form'}
    <label class="field">
      <span>{d.dir}</span>
      <!-- svelte-ignore a11y_autofocus -->
      <input class="input" type="text" bind:value={dir} autofocus spellcheck="false" />
      <small>{d.dirHint}</small>
    </label>
    <label class="field">
      <span>{d.template}</span>
      <select class="input" bind:value={template}>
        {#each ['blank', 'web-db', 'app-redis'] as const as k (k)}
          <option value={k}>{d.templates[k]}</option>
        {/each}
      </select>
    </label>
  {:else if step === 'preview' && plan}
    <p>{d.previewBody(plan.path)}</p>
    <pre class="code">{plan.content}</pre>
  {:else if step === 'done' && plan}
    <p>{d.done(plan.path)}</p>
  {/if}
  {#if error}<p class="error" role="alert">{error}</p>{/if}

  {#snippet footer()}
    {#if step === 'preview'}
      <button type="button" class="btn" onclick={() => (step = 'form')}>{d.back}</button>
    {/if}
    <span class="spacer"></span>
    <button type="button" class="btn" onclick={onclose}
      >{step === 'done' ? d.close : t.common.cancel}</button
    >
    {#if step === 'form'}
      <button type="button" class="btn btn-primary" onclick={() => void preview()}>{d.next}</button>
    {:else if step === 'preview'}
      <button type="button" class="btn btn-primary" disabled={busy} onclick={() => void create()}
        >{d.create}</button
      >
    {:else}
      <button type="button" class="btn" onclick={edit}>{d.edit}</button>
      <button type="button" class="btn btn-primary" onclick={() => void up()}>{d.up}</button>
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

  .code {
    max-height: 280px;
    margin: 0;
    padding: 8px 10px;
    overflow: auto;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    font-family: var(--font-mono);
    font-size: var(--text-sm);
    white-space: pre;
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
