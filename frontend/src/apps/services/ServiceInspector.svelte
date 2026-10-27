<script lang="ts">
  import { untrack } from 'svelte'

  import { toUserError } from '../../lib/api/errors'
  import type { ActionPlan, Journal, ServiceAction, Unit, UnitStatus } from '../../lib/api/services'
  import { formatBytes, formatDuration } from '../../lib/format'
  import { t } from '../../lib/i18n/vi'
  import Icon from '../../lib/ui/Icon.svelte'
  import { openFileInEditor } from '../../lib/wm/intents.svelte'
  import { backend } from './backend'
  import { lineLevel } from '../../lib/logLevel'
  import ConfirmCommand from '../../lib/ui/ConfirmCommand.svelte'
  import NeedsRoot from '../../lib/ui/NeedsRoot.svelte'
  import { canDo, health, shortName } from './units'

  interface Props {
    serverID: string
    unit: Unit
    /** Something was done to the unit; reload the list. */
    onchanged: () => void
  }

  let { serverID, unit, onchanged }: Props = $props()

  let status = $state.raw<UnitStatus | null>(null)
  let journal = $state.raw<Journal | null>(null)
  let error = $state('')
  let logError = $state('')
  let busy = $state<ServiceAction | null>(null)
  let confirming = $state.raw<ActionPlan | null>(null)
  // Only stop and restart are confirmed; start runs right away.
  const confirmKind = $derived(confirming?.action === 'stop' ? 'stop' : 'restart')
  /** The server refused for lack of root: what to run with sudo instead. */
  let needsRoot = $state('')
  let flash = $state<{ text: string; ok: boolean } | null>(null)
  let logBox = $state<HTMLPreElement>()

  const actions: ServiceAction[] = ['start', 'stop', 'restart']
  // The list may be fresher than the status after an action elsewhere.
  const active = $derived(status?.active ?? unit.active)

  // A reloaded list brings a new object for the same unit; only a change
  // of unit starts over.
  const name = $derived(unit.name)

  $effect(() => {
    const n = name
    untrack(() => reset(n))
  })

  function reset(name: string) {
    status = null
    journal = null
    error = ''
    needsRoot = ''
    flash = null
    void load(name)
  }

  async function load(name: string) {
    await Promise.all([loadStatus(name), loadJournal(name)])
  }

  async function loadStatus(name: string) {
    try {
      const s = await backend.status(serverID, name)
      if (unit.name === name) status = s
    } catch (e) {
      if (unit.name === name) error = toUserError(e).message
    }
  }

  async function loadJournal(name: string) {
    logError = ''
    try {
      const j = await backend.journal(serverID, name)
      if (unit.name !== name) return
      journal = j
      // Newest lines are at the bottom, where the reason for a failure is.
      requestAnimationFrame(() => logBox && (logBox.scrollTop = logBox.scrollHeight))
    } catch (e) {
      if (unit.name === name) logError = toUserError(e).message
    }
  }

  async function ask(action: ServiceAction) {
    try {
      const plan = await backend.plan(unit.name, action)
      if (plan.confirm) confirming = plan
      else await run(plan)
    } catch (e) {
      flash = { text: toUserError(e).message, ok: false }
    }
  }

  async function run(plan: ActionPlan) {
    confirming = null
    busy = plan.action
    needsRoot = ''
    flash = null
    const name = plan.unit
    try {
      await backend.act(serverID, name, plan.action)
      if (unit.name === name)
        flash = { text: t.services.done[plan.action](shortName(name)), ok: true }
    } catch (e) {
      const ue = toUserError(e)
      if (unit.name !== name) return
      if (ue.code === 'CMD_NEEDS_ROOT') needsRoot = plan.sudo
      else flash = { text: `${ue.message} ${t.services.failedHint}`, ok: false }
    } finally {
      busy = null
    }
    onchanged()
    if (unit.name === name) await load(name)
  }
</script>

<section class="inspector" aria-label={shortName(unit.name)}>
  <header>
    <span class="dot {health(active)}" aria-hidden="true"></span>
    <div class="heading">
      <h3 title={unit.name}>{shortName(unit.name)}</h3>
      <p class="desc">{status?.description || unit.description}</p>
    </div>
  </header>

  <p class="state {health(active)}">
    {t.services.states[active] ?? active}
    {#if status && status.activeFor > 0}
      <span class="muted">· {t.services.runningFor(formatDuration(status.activeFor, t.units))}</span
      >
    {/if}
  </p>

  <div class="actions">
    {#each actions as a (a)}
      <button
        type="button"
        class="btn"
        disabled={busy !== null || !canDo(active, a)}
        onclick={() => void ask(a)}
      >
        {busy === a ? t.services.working[a] : t.services.actions[a]}
      </button>
    {/each}
  </div>

  {#if needsRoot}
    <NeedsRoot {serverID} command={needsRoot} />
  {:else if flash}
    <p class="flash" class:ok={flash.ok} role="status">{flash.text}</p>
  {/if}

  {#if error}
    <p class="flash" role="alert">{error}</p>
  {:else if status}
    <dl class="facts">
      <dt>{t.services.enabled}</dt>
      <dd>{t.services.enabledStates[status.enabled] ?? (status.enabled || t.services.none)}</dd>
      <dt>{t.services.pid}</dt>
      <dd class="num">{status.mainPID || t.services.none}</dd>
      <dt>{t.services.memory}</dt>
      <dd class="num">{status.memory >= 0 ? formatBytes(status.memory) : t.services.none}</dd>
      {#if status.restarts > 0}
        <dt>{t.services.restarts}</dt>
        <dd class="num">{status.restarts}</dd>
      {/if}
      {#if status.result && status.result !== 'success'}
        <dt>{t.services.result}</dt>
        <dd class="bad">{t.services.results[status.result] ?? status.result}</dd>
      {/if}
      {#if status.file}
        <dt>{t.services.unitFile}</dt>
        <dd>
          <button
            type="button"
            class="link"
            title={t.services.openFile}
            onclick={() => openFileInEditor(serverID, status!.file)}>{status.file}</button
          >
        </dd>
      {/if}
    </dl>
  {/if}

  <div class="log-head">
    <h4>{t.services.log}</h4>
    <button
      type="button"
      class="icon-btn"
      aria-label={t.services.refreshLog}
      title={t.services.refreshLog}
      onclick={() => void loadJournal(unit.name)}
    >
      <Icon name="refresh" size={13} strokeWidth={2.2} />
    </button>
  </div>
  {#if logError}
    <p class="flash" role="alert">{logError}</p>
  {:else if !journal}
    <p class="muted small">{t.services.logLoading}</p>
  {:else if journal.restricted}
    <NeedsRoot {serverID} command={journal.sudo} message={t.services.logRestricted} />
  {:else if journal.lines.length === 0}
    <p class="muted small">{t.services.logEmpty}</p>
  {:else}
    <!-- Server output is shown as text only (docs/SECURITY.md). -->
    <pre class="log" bind:this={logBox}>{#each journal.lines as line, i (i)}<span
          class="line {lineLevel(line)}">{line}</span
        >{/each}</pre>
  {/if}
</section>

<ConfirmCommand
  command={confirming?.command ?? null}
  title={confirming ? t.services.confirmTitle[confirmKind](shortName(confirming.unit)) : ''}
  body={t.services.confirmBody[confirmKind]}
  action={t.services.actions[confirmKind]}
  onclose={() => (confirming = null)}
  onconfirm={() => confirming && void run(confirming)}
/>

<style>
  .inspector {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 14px 16px 16px;
    overflow-y: auto;
  }

  header {
    display: flex;
    align-items: flex-start;
    gap: 10px;
  }

  .heading {
    min-width: 0;
  }

  h3 {
    margin: 0;
    font-size: var(--text-lg);
    font-weight: 600;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .desc {
    margin: 2px 0 0;
    font-size: var(--text-sm);
    color: var(--fg-muted);
  }

  .dot {
    width: 10px;
    height: 10px;
    margin-top: 6px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .dot.ok {
    background: var(--ok);
  }

  .dot.busy {
    background: var(--warn);
  }

  .dot.failed {
    background: var(--danger);
  }

  .dot.off {
    background: var(--offline);
  }

  .state {
    margin: 0;
    font-weight: 600;
  }

  .state.ok {
    color: var(--ok);
  }

  .state.busy {
    color: var(--warn);
  }

  .state.failed {
    color: var(--danger);
  }

  .state.off {
    color: var(--fg-muted);
  }

  .muted {
    color: var(--fg-muted);
    font-weight: 400;
  }

  .small {
    margin: 0;
    font-size: var(--text-sm);
  }

  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .flash {
    margin: 0;
    font-size: var(--text-sm);
    color: var(--danger);
  }

  .flash.ok {
    color: var(--ok);
  }

  .facts {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: 5px 14px;
    margin: 0;
    font-size: var(--text-sm);
  }

  dt {
    color: var(--fg-muted);
  }

  dd {
    margin: 0;
    min-width: 0;
  }

  .num {
    font-variant-numeric: tabular-nums;
  }

  .bad {
    color: var(--danger);
  }

  .link {
    max-width: 100%;
    padding: 0;
    border: 0;
    background: none;
    color: var(--accent);
    font: inherit;
    text-align: left;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .link:hover {
    text-decoration: underline;
  }

  .log-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-top: 4px;
  }

  h4 {
    margin: 0;
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--fg-muted);
  }

  .icon-btn {
    display: flex;
    padding: 4px;
    border: 0;
    border-radius: 50%;
    background: none;
    color: var(--fg-muted);
  }

  .icon-btn:hover {
    background: var(--fill-soft);
  }

  .log {
    flex: 1;
    min-height: 120px;
    margin: 0;
    padding: 8px 10px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    line-height: 1.5;
    white-space: pre-wrap;
    word-break: break-word;
    overflow-y: auto;
    user-select: text;
  }

  /* One block per line, so copying keeps the line breaks. */
  .line {
    display: block;
  }

  .log .error {
    color: var(--danger);
  }

  .log .warn {
    color: var(--warn);
  }
</style>
