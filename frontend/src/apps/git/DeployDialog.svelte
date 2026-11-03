<script lang="ts">
  import { untrack } from 'svelte'

  import { toUserError } from '../../lib/api/errors'
  import type { DeployLine } from '../../lib/api/git'
  import { t } from '../../lib/i18n'
  import { deploys } from '../../lib/stores/deploys.svelte'
  import Dialog from '../../lib/ui/Dialog.svelte'
  import { openTerminalAt } from '../../lib/wm/intents.svelte'
  import { backend } from './backend'
  import { hasRecipe, loadRecipe, saveRecipe } from './recipe'

  interface Props {
    serverID: string
    dir: string
    /** Steps guessed from the repository, until a recipe is saved. */
    suggested: string[]
    open: boolean
    onclose: () => void
  }

  let { serverID, dir, suggested, open, onclose }: Props = $props()
  const g = t.git.deploy

  let steps = $state('')
  let command = $state<DeployLine>()
  let error = $state('')

  $effect(() => {
    if (!open) return
    steps = loadRecipe(serverID, dir, suggested).join('\n')
    command = undefined
    error = ''
    // A saved recipe goes straight to its command: one more click deploys.
    if (hasRecipe(serverID, dir)) untrack(() => void preview())
  })

  const list = $derived(steps.split('\n'))

  async function preview() {
    error = ''
    try {
      command = await backend.deployLine(dir, list)
      saveRecipe(serverID, dir, list)
    } catch (e) {
      error = toUserError(e).message
    }
  }

  function run() {
    if (!command) return
    deploys.watch(serverID, dir, command.nonce, command.steps)
    openTerminalAt(serverID, undefined, command.line, command.nonce)
    onclose()
  }
</script>

<Dialog {open} title={g.title} width={560} {onclose}>
  <p class="muted">{g.body(dir)}</p>
  <textarea
    class="input steps"
    rows="7"
    bind:value={steps}
    oninput={() => (command = undefined)}
    spellcheck="false"></textarea>
  {#if command}
    <p>{g.commandIntro}</p>
    <pre class="command">{command.line}</pre>
  {/if}
  {#if error}<p class="error" role="alert">{error}</p>{/if}

  {#snippet footer()}
    <span class="spacer"></span>
    <button type="button" class="btn" onclick={onclose}>{t.common.cancel}</button>
    {#if command}
      <button type="button" class="btn btn-primary" onclick={run}>{g.run}</button>
    {:else}
      <button type="button" class="btn btn-primary" onclick={() => void preview()}>{g.next}</button>
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

  .steps {
    box-sizing: border-box;
    width: 100%;
    margin: 0 0 10px;
    font-family: var(--font-mono);
    resize: vertical;
  }

  .command {
    margin: 0;
    padding: 8px 10px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    font-family: var(--font-mono);
    font-size: var(--text-sm);
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
