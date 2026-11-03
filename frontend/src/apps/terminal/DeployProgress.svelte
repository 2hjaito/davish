<script lang="ts">
  import { formatElapsed } from '../../lib/format'
  import { t } from '../../lib/i18n'
  import { type DeployRun, durations } from '../../lib/stores/deploys.svelte'
  import Icon from '../../lib/ui/Icon.svelte'

  interface Props {
    run: DeployRun
    onhide: () => void
  }

  let { run, onhide }: Props = $props()
  const d = t.deploys

  // Ticks while the run goes, for the running step's time.
  let now = $state(Date.now())
  $effect(() => {
    if (run.state !== 'running') return
    const id = setInterval(() => (now = Date.now()), 1000)
    return () => clearInterval(id)
  })

  const end = $derived(run.endedAt ?? now)
  const took = $derived(durations(run, end))
  const total = $derived(run.startedAt === undefined ? 0 : end - run.startedAt)
  const step = $derived(run.steps[run.current - 1] ?? '')

  const icon = $derived(
    run.state === 'ok'
      ? 'check'
      : run.state === 'failed'
        ? 'warning'
        : run.state === 'stopped'
          ? 'stop'
          : 'clock',
  )

  const label = $derived.by(() => {
    switch (run.state) {
      case 'waiting':
        return d.waiting
      case 'running':
        return d.running(Math.max(run.current, 1), run.steps.length)
      case 'ok':
        return d.ok
      case 'failed':
        return d.failed(run.current)
      case 'stopped':
        return d.stopped
    }
  })

  /** done, current, failed or pending, for step i (1-based). */
  function segment(i: number): string {
    if (i < run.current || (i === run.current && run.state === 'ok')) return 'done'
    if (i === run.current) return run.state === 'running' ? 'current' : 'failed'
    return 'pending'
  }
</script>

<div class="progress glass-dark {run.state}" role="status" aria-label={d.progress}>
  <span class="icon"><Icon name={icon} size={13} /></span>
  <span class="label">{label}</span>
  {#if step && run.state !== 'ok'}
    <code class="step" title={step}>{step}</code>
  {/if}
  {#if run.state !== 'waiting'}
    <span class="time">{formatElapsed(total)}</span>
  {/if}
  <ol class="segments">
    {#each run.steps as s, i (i)}
      <li
        class={segment(i + 1)}
        title={took[i] === undefined ? s : `${s} · ${formatElapsed(took[i] ?? 0)}`}
      ></li>
    {/each}
  </ol>
  {#if run.state !== 'running' && run.state !== 'waiting'}
    <button class="hide" aria-label={d.hide} onclick={onhide}>
      <Icon name="close" size={10} strokeWidth={2.5} />
    </button>
  {/if}
</div>

<style>
  .progress {
    position: absolute;
    top: 10px;
    right: 14px;
    z-index: 1;
    display: flex;
    align-items: center;
    gap: 8px;
    max-width: calc(100% - 28px);
    padding: 6px 10px;
    border-radius: var(--control-radius);
    color: var(--fg-on-dark);
    font-size: 12px;
    white-space: nowrap;
  }

  .icon {
    display: flex;
    color: var(--fg-on-dark-muted);
  }

  .ok .icon {
    color: var(--ok-on-dark);
  }

  .failed .icon {
    color: var(--danger-on-dark);
  }

  .stopped .icon {
    color: var(--warn-on-dark);
  }

  .step {
    min-width: 0;
    max-width: 260px;
    overflow: hidden;
    text-overflow: ellipsis;
    color: var(--fg-on-dark-muted);
    font-family: var(--font-mono);
  }

  .time {
    color: var(--fg-on-dark-muted);
    font-variant-numeric: tabular-nums;
  }

  .segments {
    display: flex;
    gap: 3px;
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .segments li {
    width: 14px;
    height: 4px;
    border-radius: 2px;
    background: var(--term-scrollbar);
  }

  .segments .done {
    background: var(--ok-on-dark);
  }

  .segments .current {
    background: var(--accent);
  }

  .segments .failed {
    background: var(--danger-on-dark);
  }

  .hide {
    display: flex;
    padding: 3px;
    border: none;
    border-radius: 50%;
    background: transparent;
    color: var(--fg-on-dark-muted);
    cursor: pointer;
  }

  .hide:hover,
  .hide:focus-visible {
    background: var(--term-scrollbar);
    color: var(--fg-on-dark);
  }
</style>
