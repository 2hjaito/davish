<script lang="ts">
  import { untrack } from 'svelte'

  import type {
    ActionPlan,
    Container,
    ContainerAction,
    ContainerDetail,
    ContainerStats,
    LogLine,
  } from '../../lib/api/docker'
  import { toUserError } from '../../lib/api/errors'
  import { formatBytes, formatDuration, formatPct } from '../../lib/format'
  import { t } from '../../lib/i18n/vi'
  import { lineLevel } from '../../lib/logLevel'
  import ConfirmCommand from '../../lib/ui/ConfirmCommand.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import NeedsRoot from '../../lib/ui/NeedsRoot.svelte'
  import { openLogs } from '../../lib/wm/intents.svelte'
  import { backend } from './backend'
  import { canDo, clock, health, since } from './containers'

  interface Props {
    serverID: string
    container: Container
    /** Its latest sample, when running and measured. */
    stats: ContainerStats | undefined
    /** Something was done to the container; reload the list. */
    onchanged: () => void
  }

  let { serverID, container, stats, onchanged }: Props = $props()

  let detail = $state.raw<ContainerDetail | null>(null)
  let logs = $state.raw<LogLine[] | null>(null)
  let error = $state('')
  let logError = $state('')
  let tab = $state<'logs' | 'inspect'>('logs')
  let busy = $state<ContainerAction | null>(null)
  let confirming = $state.raw<ActionPlan | null>(null)
  const confirmKind = $derived(confirming?.action === 'stop' ? 'stop' : 'restart')
  /** The server refused for lack of root: what to run with sudo instead. */
  let needsRoot = $state('')
  let flash = $state<{ text: string; ok: boolean } | null>(null)
  let logBox = $state<HTMLElement>()

  const actions: ContainerAction[] = ['start', 'stop', 'restart']
  const status = $derived(detail?.state ?? container.state)
  // A reloaded list brings a new object for the same container; only a
  // change of container starts over.
  const name = $derived(container.name)

  $effect(() => {
    const n = name
    untrack(() => reset(n))
  })

  function reset(n: string) {
    detail = null
    logs = null
    error = ''
    needsRoot = ''
    flash = null
    void load(n)
  }

  async function load(n: string) {
    await Promise.all([loadDetail(n), loadLogs(n)])
  }

  async function loadDetail(n: string) {
    try {
      const d = await backend.inspect(serverID, n)
      if (name === n) detail = d
    } catch (e) {
      if (name === n) error = toUserError(e).message
    }
  }

  async function loadLogs(n: string) {
    logError = ''
    try {
      const l = await backend.logs(serverID, n)
      if (name !== n) return
      logs = l
      requestAnimationFrame(() => logBox && (logBox.scrollTop = logBox.scrollHeight))
    } catch (e) {
      if (name === n) logError = toUserError(e).message
    }
  }

  async function ask(action: ContainerAction) {
    try {
      const plan = await backend.plan(name, action)
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
    const n = plan.name
    try {
      await backend.act(serverID, n, plan.action)
      if (name === n) flash = { text: t.docker.done[plan.action](n), ok: true }
    } catch (e) {
      const ue = toUserError(e)
      if (name !== n) return
      if (ue.code === 'CMD_NEEDS_ROOT') needsRoot = plan.sudo
      else flash = { text: `${ue.message} ${t.docker.failedHint}`, ok: false }
    } finally {
      busy = null
    }
    onchanged()
    if (name === n) await load(n)
  }

  function stateLine(d: ContainerDetail | null): string {
    if (!d) return ''
    if (d.state === 'running' && d.startedAt)
      return t.docker.runningFor(formatDuration(since(d.startedAt, Date.now()), t.units))
    if (d.state === 'restarting') return t.docker.restartsN(d.restarts)
    if (d.state === 'exited' || d.state === 'dead') return t.docker.exitCode(d.exitCode)
    return ''
  }
</script>

<section class="inspector" aria-label={name}>
  <header>
    <span class="dot {health(status)}" aria-hidden="true"></span>
    <div class="heading">
      <h3 title={container.id}>{name}</h3>
      <p class="desc">{container.image}</p>
    </div>
  </header>

  <p class="state {health(status)}">
    {t.docker.states[status] ?? status}
    {#if stateLine(detail)}<span class="muted">· {stateLine(detail)}</span>{/if}
  </p>

  <div class="actions">
    {#each actions as a (a)}
      <button
        type="button"
        class="btn"
        disabled={busy !== null || !canDo(status, a)}
        onclick={() => void ask(a)}
      >
        {busy === a ? t.docker.working[a] : t.docker.actions[a]}
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
  {:else if detail}
    <dl class="facts">
      {#if container.state === 'running'}
        <dt>{t.docker.cpu}</dt>
        <dd class="num">{stats ? formatPct(stats.cpu) : t.docker.statsLoading}</dd>
        <dt>{t.docker.memory}</dt>
        <dd class="num">
          {stats
            ? `${formatBytes(stats.memUsed)} / ${formatBytes(stats.memLimit)}`
            : t.docker.statsLoading}
        </dd>
      {/if}
      <dt>{t.docker.policy}</dt>
      <dd>{t.docker.policies[detail.policy] ?? (detail.policy || t.docker.none)}</dd>
      {#if detail.restarts > 0}
        <dt>{t.docker.restarts}</dt>
        <dd class="num bad">{detail.restarts}</dd>
      {/if}
      {#if detail.health}
        <dt>{t.docker.health}</dt>
        <dd class:bad={detail.health === 'unhealthy'}>
          {t.docker.healths[detail.health] ?? detail.health}
        </dd>
      {/if}
      {#if detail.ports.length > 0}
        <dt>{t.docker.ports}</dt>
        <dd class="mono">{detail.ports.join('\n')}</dd>
      {/if}
      {#if detail.mounts.length > 0}
        <dt>{t.docker.mounts}</dt>
        <dd class="mono">{detail.mounts.join('\n')}</dd>
      {/if}
      {#if detail.oomKilled}
        <dt>{t.docker.error}</dt>
        <dd class="bad">{t.docker.oom}</dd>
      {:else if detail.error}
        <dt>{t.docker.error}</dt>
        <dd class="bad">{detail.error}</dd>
      {/if}
    </dl>
  {/if}

  <div class="tabs-row">
    <div class="segmented" role="group" aria-label={t.docker.tabsLabel}>
      <button aria-pressed={tab === 'logs'} onclick={() => (tab = 'logs')}>
        {t.docker.tabs.logs}
      </button>
      <button aria-pressed={tab === 'inspect'} onclick={() => (tab = 'inspect')}>
        {t.docker.tabs.inspect}
      </button>
    </div>
    {#if tab === 'logs'}
      <button
        type="button"
        class="icon-btn"
        aria-label={t.docker.refreshLog}
        title={t.docker.refreshLog}
        onclick={() => void loadLogs(name)}
      >
        <Icon name="refresh" size={13} strokeWidth={2.2} />
      </button>
      <button
        type="button"
        class="icon-btn"
        aria-label={t.logs.openInLogs}
        title={t.logs.openInLogs}
        onclick={() => openLogs(serverID, { kind: 'docker', target: name })}
      >
        <Icon name="logs" size={13} strokeWidth={2.2} />
      </button>
    {/if}
  </div>

  <!-- Server output is shown as text only (docs/SECURITY.md). -->
  {#if tab === 'inspect'}
    {#if detail}<pre class="box">{detail.json}</pre>{/if}
  {:else if logError}
    <p class="flash" role="alert">{logError}</p>
  {:else if !logs}
    <p class="muted small">{t.docker.logLoading}</p>
  {:else if logs.length === 0}
    <p class="muted small">{t.docker.logEmpty}</p>
  {:else}
    <pre class="box" bind:this={logBox}>{#each logs as l, i (i)}<span
          class="line {lineLevel(l.text)}"
          title={l.stderr ? `${l.time} · ${t.docker.stderr}` : l.time}
          ><span class="time" class:err={l.stderr}>{clock(l.time)}</span>{l.text}</span
        >{/each}</pre>
  {/if}
</section>

<ConfirmCommand
  command={confirming?.command ?? null}
  title={confirming ? t.docker.confirmTitle[confirmKind](confirming.name) : ''}
  body={t.docker.confirmBody[confirmKind]}
  action={t.docker.actions[confirmKind]}
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
    font-family: var(--font-mono);
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
    overflow-wrap: anywhere;
  }

  .num {
    font-variant-numeric: tabular-nums;
  }

  .mono {
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    white-space: pre-line;
  }

  .bad {
    color: var(--danger);
  }

  .tabs-row {
    display: flex;
    align-items: center;
    gap: 2px;
    margin-top: 4px;
  }

  .segmented {
    display: flex;
    margin-right: auto;
    padding: 2px;
    border-radius: 8px;
    background: var(--fill-soft);
  }

  .segmented button {
    min-width: 56px;
    padding: 2px 10px;
    border: 0;
    border-radius: 6px;
    background: none;
    font-size: var(--text-xs);
    color: var(--fg-muted);
    transition: background var(--dur-fast) var(--ease-out);
  }

  .segmented button[aria-pressed='true'] {
    background: var(--chart-surface);
    color: var(--fg);
    font-weight: 600;
    box-shadow: var(--switch-knob-shadow);
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

  .box {
    flex: 1;
    min-height: 140px;
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

  .time {
    margin-right: 8px;
    color: var(--fg-subtle);
    user-select: none;
  }

  /* stderr is marked on the time, not the text: plenty of programs log
     normal output there. */
  .time.err {
    color: var(--warn);
  }

  .line.error {
    color: var(--danger);
  }

  .line.warn {
    color: var(--warn);
  }
</style>
