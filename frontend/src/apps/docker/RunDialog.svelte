<script lang="ts">
  import type { Mount, PortMap, RunPlan, RunSpec } from '../../lib/api/docker'
  import { toUserError } from '../../lib/api/errors'
  import { t } from '../../lib/i18n'
  import Dialog from '../../lib/ui/Dialog.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import { openTerminalAt } from '../../lib/wm/intents.svelte'
  import { backend } from './backend'

  interface Props {
    serverID: string
    /** pull asks only for an image; the dialog is closed while null. */
    mode: 'pull' | 'run' | null
    /** Prefills the image. */
    image?: string
    onclose: () => void
  }

  let { serverID, mode, image: initial = '', onclose }: Props = $props()
  const d = t.docker.run

  let step = $state<'form' | 'preview'>('form')
  let name = $state('')
  let image = $state('')
  let ports = $state<PortMap[]>([])
  let env = $state('')
  let envFile = $state('')
  let mounts = $state<Mount[]>([])
  let restart = $state<RunSpec['restart']>('unless-stopped')
  let network = $state('')
  let plan = $state.raw<RunPlan | null>(null)
  let command = $state('')
  let error = $state('')

  $effect(() => {
    if (!mode) return
    step = 'form'
    image = initial
    error = ''
    plan = null
  })

  const spec = $derived<RunSpec>({
    name: name.trim(),
    image: image.trim(),
    ports,
    env: env.split('\n'),
    envFile: envFile.trim(),
    mounts,
    restart,
    network: network.trim(),
    args: [],
  })

  async function preview() {
    error = ''
    try {
      if (mode === 'pull') {
        command = await backend.pullCommand(spec.image)
        plan = null
      } else {
        plan = await backend.planRun(spec)
        command = plan.command
      }
      step = 'preview'
    } catch (e) {
      error = toUserError(e).message
    }
  }

  function open() {
    openTerminalAt(serverID, undefined, command)
    onclose()
  }
</script>

<Dialog open={mode !== null} title={mode === 'pull' ? d.pullTitle : d.title} width={560} {onclose}>
  {#if step === 'form'}
    <label class="field">
      <span>{d.image}</span>
      <!-- svelte-ignore a11y_autofocus -->
      <input
        class="input"
        type="text"
        bind:value={image}
        placeholder="nginx:1.27"
        autofocus
        spellcheck="false"
      />
      {#if mode === 'pull'}<small>{d.imageHint}</small>{/if}
    </label>
    {#if mode === 'run'}
      <div class="row">
        <label class="field">
          <span>{d.name}</span>
          <input class="input" type="text" bind:value={name} spellcheck="false" />
        </label>
        <label class="field">
          <span>{d.restart}</span>
          <select class="input" bind:value={restart}>
            {#each ['unless-stopped', 'always', 'on-failure', 'no'] as const as r (r)}
              <option value={r}>{d.restarts[r]}</option>
            {/each}
          </select>
        </label>
      </div>

      <fieldset>
        <legend>{d.ports}</legend>
        {#each ports as p, i (i)}
          <div class="line">
            <input class="input addr" type="text" bind:value={p.host} placeholder="127.0.0.1" />
            <input class="input num" type="number" bind:value={p.hostPort} placeholder="8080" />
            <span>→</span>
            <input class="input num" type="number" bind:value={p.containerPort} placeholder="80" />
            <label class="check"><input type="checkbox" bind:checked={p.udp} /> UDP</label>
            <button
              type="button"
              class="tool"
              aria-label={d.removeRow}
              onclick={() => ports.splice(i, 1)}><Icon name="minus" size={13} /></button
            >
          </div>
        {/each}
        <button
          type="button"
          class="link"
          onclick={() => ports.push({ host: '', hostPort: 0, containerPort: 0, udp: false })}
          >{d.addPort}</button
        >
        <small>{d.portsHint}</small>
      </fieldset>

      <fieldset>
        <legend>{d.mounts}</legend>
        {#each mounts as m, i (i)}
          <div class="line">
            <input class="input" type="text" bind:value={m.source} placeholder={d.source} />
            <span>→</span>
            <input class="input" type="text" bind:value={m.target} placeholder="/data" />
            <label class="check"><input type="checkbox" bind:checked={m.readOnly} /> {d.ro}</label>
            <button
              type="button"
              class="tool"
              aria-label={d.removeRow}
              onclick={() => mounts.splice(i, 1)}><Icon name="minus" size={13} /></button
            >
          </div>
        {/each}
        <button
          type="button"
          class="link"
          onclick={() => mounts.push({ source: '', target: '', readOnly: false })}
          >{d.addMount}</button
        >
      </fieldset>

      <label class="field">
        <span>{d.env}</span>
        <textarea
          class="input mono"
          rows="3"
          bind:value={env}
          placeholder="KEY=value"
          spellcheck="false"></textarea>
      </label>
      <div class="row">
        <label class="field">
          <span>{d.envFile}</span>
          <input class="input" type="text" bind:value={envFile} placeholder="/srv/app/.env" />
        </label>
        <label class="field">
          <span>{d.network}</span>
          <input class="input" type="text" bind:value={network} placeholder="bridge" />
        </label>
      </div>
    {/if}
  {:else}
    {#if plan && plan.public.length > 0}
      <p class="warn"><Icon name="warning" size={14} /> {d.publicWarn(plan.public.join(', '))}</p>
    {/if}
    <p>{d.commandIntro}</p>
    <pre class="command">{command}</pre>
  {/if}
  {#if error}<p class="error" role="alert">{error}</p>{/if}

  {#snippet footer()}
    {#if step === 'preview'}
      <button type="button" class="btn" onclick={() => (step = 'form')}>{d.back}</button>
    {/if}
    <span class="spacer"></span>
    <button type="button" class="btn" onclick={onclose}>{t.common.cancel}</button>
    {#if step === 'form'}
      <button
        type="button"
        class="btn btn-primary"
        disabled={!spec.image}
        onclick={() => void preview()}>{d.next}</button
      >
    {:else}
      <button type="button" class="btn btn-primary" onclick={open}>{d.open}</button>
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
    flex: 1;
    flex-direction: column;
    gap: 4px;
    margin: 0 0 12px;
  }

  .row {
    display: flex;
    gap: 12px;
  }

  fieldset {
    display: flex;
    flex-direction: column;
    gap: 6px;
    margin: 0 0 12px;
    padding: 0;
    border: none;
  }

  legend {
    margin-bottom: 4px;
    padding: 0;
  }

  .line {
    display: flex;
    gap: 6px;
    align-items: center;
  }

  .line .input {
    flex: 1;
    min-width: 0;
  }

  .line .num {
    flex: 0 0 80px;
  }

  .line .addr {
    flex: 0 0 110px;
  }

  .check {
    display: flex;
    gap: 4px;
    align-items: center;
    white-space: nowrap;
  }

  .link {
    align-self: flex-start;
  }

  small {
    color: var(--fg-muted);
    font-size: var(--text-xs);
  }

  .mono {
    font-family: var(--font-mono);
    resize: vertical;
  }

  .warn {
    display: flex;
    gap: 6px;
    align-items: flex-start;
    color: var(--warn);
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
