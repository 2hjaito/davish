<script lang="ts">
  import { tick, untrack } from 'svelte'

  import { toUserError } from '../../lib/api/errors'
  import type { DeployInfo } from '../../lib/api/git'
  import { t } from '../../lib/i18n'
  import { deploys } from '../../lib/stores/deploys.svelte'
  import Dialog from '../../lib/ui/Dialog.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import { openTerminalAt } from '../../lib/wm/intents.svelte'
  import { backend } from './backend'
  import { type DeployPlan, loadPlan, runnerKey, savePlan, stepsOf } from './recipe'

  interface Props {
    serverID: string
    dir: string
    open: boolean
    onclose: () => void
  }

  let { serverID, dir, open, onclose }: Props = $props()
  const g = t.git.deploy

  let info = $state.raw<DeployInfo | null>(null)
  let plan = $state<DeployPlan>({ pull: true, install: false, build: '', restart: '', custom: [] })
  let lostRestart = $state('')
  let loading = $state(false)
  let running = $state(false)
  let error = $state('')

  const repo = $derived(dir.split('/').filter(Boolean).pop() ?? dir)

  async function look() {
    loading = true
    error = ''
    info = null
    try {
      const found = await backend.deployInfo(serverID, dir)
      const l = loadPlan(serverID, dir, found)
      plan = l.plan
      lostRestart = l.lostRestart
      info = found
    } catch (e) {
      error = toUserError(e).message
    } finally {
      loading = false
    }
  }

  $effect(() => {
    if (open) untrack(() => void look())
  })

  const steps = $derived(
    info
      ? stepsOf(plan, info, {
          pull: g.pull,
          install: g.install,
          build: g.buildStep,
          restart: g.restartStep,
        })
      : [],
  )
  const runner = $derived(info?.runners.find((r) => runnerKey(r) === plan.restart))
  const build = $derived(info?.builds.find((b) => b.name === plan.build))
  const sudo = $derived(
    (runner?.sudo ?? false) || plan.custom.some((c) => /(^|[\s;&|(])sudo\s/.test(c)),
  )

  function toggleBuild(on: boolean) {
    const first = info?.builds[0]
    plan.build = on && first ? first.name : ''
  }

  // The custom step inputs, to put the cursor in a new one.
  const inputs: HTMLInputElement[] = []

  async function addCustom() {
    plan.custom.push('')
    await tick()
    inputs[plan.custom.length - 1]?.focus()
  }

  function toggleRestart(on: boolean) {
    const first = info?.runners[0]
    plan.restart = on && first ? runnerKey(first) : ''
  }

  async function deploy() {
    if (steps.length === 0) return
    running = true
    error = ''
    try {
      const d = await backend.deployLine(dir, steps)
      savePlan(serverID, dir, $state.snapshot(plan))
      deploys.watch(serverID, dir, d.nonce, d.steps, !d.sudo)
      openTerminalAt(serverID, undefined, d.line, d.nonce, !d.sudo)
      onclose()
    } catch (e) {
      error = toUserError(e).message
    } finally {
      running = false
    }
  }
</script>

<Dialog {open} title={g.title(repo)} width={600} {onclose}>
  {#if loading}
    <p class="muted">{g.looking}</p>
  {:else if info}
    <p class="found">
      <span class="muted">{g.found}</span>
      {#if info.install}<span class="fact">Node.js ({info.install.name})</span>{/if}
      {#if info.runners[0]}
        <span class="fact">{g.runsWith(info.runners[0].kind, info.runners[0].name)}</span>
      {:else}
        <span class="fact">{g.notRunning}</span>
      {/if}
    </p>
    {#if lostRestart}<p class="warn" role="alert">{g.lostRestart(lostRestart)}</p>{/if}

    <ol class="steps">
      <li>
        <label class="step">
          <input type="checkbox" class="switch" role="switch" bind:checked={plan.pull} />
          <span class="what">
            <strong>{g.pull}</strong>
            <code>{info.pull.command}</code>
          </span>
        </label>
      </li>

      {#if info.install}
        <li>
          <label class="step">
            <input type="checkbox" class="switch" role="switch" bind:checked={plan.install} />
            <span class="what">
              <strong>{g.install}</strong>
              <code>{info.install.command}</code>
            </span>
          </label>
        </li>
      {/if}

      {#if info.builds.length > 0}
        <li>
          <div class="step">
            <input
              type="checkbox"
              class="switch"
              role="switch"
              aria-label={g.build}
              checked={plan.build !== ''}
              onchange={(e) => toggleBuild(e.currentTarget.checked)}
            />
            <span class="what">
              <strong>{g.build}</strong>
              {#if build}<code>{build.command}</code>{/if}
            </span>
            {#if plan.build}
              <select class="input pick" aria-label={g.build} bind:value={plan.build}>
                {#each info.builds as b (b.name)}
                  <option value={b.name}>{b.name}</option>
                {/each}
              </select>
            {/if}
          </div>
        </li>
      {/if}

      {#each plan.custom as _, i (i)}
        <li>
          <div class="step">
            <span class="slot"><Icon name="terminal" size={14} /></span>
            <span class="what">
              <strong>{g.custom}</strong>
              <input
                class="input mono"
                aria-label={g.custom}
                placeholder={g.customPlaceholder}
                spellcheck="false"
                bind:value={plan.custom[i]}
                bind:this={inputs[i]}
              />
            </span>
            <button
              type="button"
              class="remove"
              aria-label={g.removeCustom}
              onclick={() => plan.custom.splice(i, 1)}
            >
              <Icon name="trash" size={13} />
            </button>
          </div>
        </li>
      {/each}

      <li>
        <div class="step" class:off={info.runners.length === 0}>
          <input
            type="checkbox"
            class="switch"
            role="switch"
            aria-label={g.restart}
            checked={plan.restart !== ''}
            disabled={info.runners.length === 0}
            onchange={(e) => toggleRestart(e.currentTarget.checked)}
          />
          <span class="what">
            <strong>{g.restart}</strong>
            {#if runner}<code>{runner.command}</code>{/if}
          </span>
          {#if plan.restart && info.runners.length > 1}
            <select class="input pick" aria-label={g.restart} bind:value={plan.restart}>
              {#each info.runners as r (runnerKey(r))}
                <option value={runnerKey(r)}>{g.runner(r.kind, r.name)}</option>
              {/each}
            </select>
          {:else if runner}
            <span class="chosen">{g.runner(runner.kind, runner.name)}</span>
          {/if}
        </div>
      </li>
    </ol>

    <button type="button" class="btn add" onclick={() => void addCustom()}>
      <Icon name="plus" size={12} />
      {g.addCustom}
    </button>

    <div class="notes">
      {#if info.runners.length === 0}<p>{g.noRunner}</p>{/if}
      {#if runner?.kind === 'compose'}<p>{g.composeNote}</p>{/if}
      <p>{sudo ? g.sudoNote : g.runNote}</p>
      <p>{g.saved}</p>
    </div>
  {/if}
  {#if error}
    <p class="error" role="alert">{error}</p>
  {/if}

  {#snippet footer()}
    {#if error && !info}
      <button type="button" class="btn" onclick={() => void look()}>{g.retry}</button>
    {/if}
    <span class="spacer"></span>
    <button type="button" class="btn" onclick={onclose}>{t.common.cancel}</button>
    <button
      type="button"
      class="btn btn-primary"
      disabled={!info || steps.length === 0 || running}
      title={info && steps.length === 0 ? g.empty : undefined}
      onclick={() => void deploy()}
    >
      <Icon name="play" size={12} />
      {g.run}
    </button>
  {/snippet}
</Dialog>

<style>
  p {
    margin: 0 0 10px;
    line-height: 1.45;
  }

  .muted {
    color: var(--fg-muted);
  }

  .found {
    display: flex;
    flex-wrap: wrap;
    gap: 4px 8px;
    font-size: var(--text-sm);
  }

  .fact {
    padding: 1px 8px;
    border-radius: var(--item-radius);
    background: var(--fill-soft);
  }

  .warn {
    color: var(--warn);
    font-size: var(--text-sm);
  }

  .steps {
    display: flex;
    flex-direction: column;
    gap: 2px;
    margin: 0 0 8px;
    padding: 0;
    list-style: none;
  }

  .step {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 8px 10px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
  }

  label.step {
    cursor: pointer;
  }

  .step.off {
    opacity: 0.6;
  }

  /* Where the other steps have their switch. */
  .slot {
    flex: none;
    display: flex;
    justify-content: center;
    width: 32px;
    color: var(--fg-muted);
  }

  .chosen {
    flex: none;
    color: var(--fg-muted);
    font-size: var(--text-sm);
  }

  .what {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .what strong {
    font-size: var(--text-sm);
    font-weight: 600;
  }

  code {
    overflow: hidden;
    color: var(--fg-muted);
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    text-overflow: ellipsis;
    white-space: nowrap;
    user-select: text;
  }

  .what .input {
    width: 100%;
    box-sizing: border-box;
  }

  .pick {
    flex: none;
    max-width: 220px;
  }

  .remove {
    display: flex;
    padding: 6px;
    border: none;
    border-radius: var(--item-radius);
    background: transparent;
    color: var(--fg-muted);
    cursor: pointer;
  }

  .remove:hover,
  .remove:focus-visible {
    background: var(--fill-soft);
    color: var(--danger);
  }

  .add {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    margin-bottom: 12px;
  }

  .notes p {
    margin: 0 0 4px;
    color: var(--fg-muted);
    font-size: var(--text-xs);
  }

  .error {
    margin: 10px 0 0;
    color: var(--danger);
  }

  .spacer {
    flex: 1;
  }

  .btn-primary {
    display: inline-flex;
    align-items: center;
    gap: 6px;
  }
</style>
