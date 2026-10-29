<script lang="ts">
  import { untrack } from 'svelte'

  import { copyText } from '../api/clipboard'
  import { diagRun, diagSupportInfo, type DiagReport } from '../api/diag'
  import { hasRuntime } from '../api/runtime'
  import { t } from '../i18n/vi'
  import Empty from './Empty.svelte'
  import Icon from './Icon.svelte'
  import type { IconName } from './icons'

  // What an app shows until its first answer from the server: loading, a
  // server that is slow to answer, or an error. Both of the latter offer a
  // diagnosis, so a user can see why (and send it to whoever helps them)
  // without anyone logging in to the server.
  interface Props {
    serverID: string
    /** The app's error text; empty while loading. */
    error: string
    icon: IconName
    loading: string
    onretry: () => void
  }

  let { serverID, error, icon, loading, onretry }: Props = $props()

  const d = t.diag
  /** Seconds before a load counts as slow. */
  const SLOW = 8

  let slow = $state(false)
  let report = $state.raw<DiagReport | null>(null)
  let running = $state(false)
  let copied = $state(false)

  $effect(() => {
    if (error) return
    untrack(() => (slow = false))
    const timer = setTimeout(() => (slow = true), SLOW * 1000)
    return () => clearTimeout(timer)
  })

  async function diagnose() {
    running = true
    report = null
    try {
      report = await diagRun(serverID)
    } finally {
      running = false
    }
  }

  async function copy() {
    const info = hasRuntime ? await diagSupportInfo() : null
    const text = [
      `Davish ${info?.version ?? __APP_VERSION__} (${info?.os ?? navigator.platform})`,
      `${t.diag.copyError}: ${error || t.diag.slowTitle}`,
      report ? `\n${t.diag.copyDiag}:\n${report.text}` : '',
      info?.logTail ? `\n${t.diag.copyLog} (${info.logPath}):\n${info.logTail}` : '',
    ]
      .filter(Boolean)
      .join('\n')
    copied = await copyText(text)
    setTimeout(() => (copied = false), 2000)
  }
</script>

{#if !error && !slow}
  <Empty {icon} title={loading} />
{:else}
  <div class="state">
    <span class="icon" class:warn={!!error}><Icon name={error ? 'warning' : icon} size={26} /></span
    >
    <h3>{error || d.slowTitle}</h3>
    <p>{error ? d.errorHint : d.slowHint}</p>
    <div class="actions">
      <button class="btn" onclick={onretry}>{d.retry}</button>
      <button
        class="btn btn-primary"
        disabled={running || !hasRuntime}
        onclick={() => void diagnose()}>{running ? d.running : d.diagnose}</button
      >
      <button class="btn" onclick={() => void copy()}>{copied ? d.copied : d.copy}</button>
    </div>

    {#if report}
      <section class="report">
        <p class="verdict" class:ok={report.verdict === 'ok'}>
          {d.verdicts[report.verdict]?.title ?? report.verdict}
        </p>
        <p class="advice">{d.verdicts[report.verdict]?.advice ?? ''}</p>
        <ul>
          {#each report.steps as s (s.id)}
            <li>
              <span class:bad={!s.ok}>{s.ok ? '✓' : '✗'}</span>
              {d.steps[s.id] ?? s.id}
              <span class="muted">{s.ms} ms{s.detail ? ` · ${s.detail}` : ''}</span>
            </li>
          {/each}
        </ul>
        {#if report.findings?.length}
          <p class="advice">{d.findings}</p>
          <pre>{report.findings.map((f) => `${f.file}:${f.line}  ${f.text}`).join('\n')}</pre>
        {/if}
      </section>
    {/if}
  </div>
{/if}

<style>
  .state {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 8px;
    padding: 24px;
    text-align: center;
    overflow: auto;
  }

  .icon {
    color: var(--fg-subtle);
  }

  .icon.warn {
    color: var(--warn);
  }

  h3 {
    margin: 0;
    font-size: var(--text-md);
  }

  p {
    margin: 0;
    max-width: 460px;
    font-size: var(--text-sm);
    color: var(--fg-muted);
  }

  .actions {
    display: flex;
    gap: 8px;
    margin-top: 6px;
  }

  .report {
    width: min(520px, 100%);
    margin-top: 10px;
    padding: 12px 14px;
    border-radius: 12px;
    background: var(--fill-soft);
    text-align: left;
  }

  .verdict {
    margin-bottom: 4px;
    font-weight: 600;
    color: var(--danger);
  }

  .verdict.ok {
    color: var(--ok);
  }

  .advice {
    max-width: none;
    margin-bottom: 8px;
    color: var(--fg);
    line-height: 1.45;
  }

  ul {
    list-style: none;
    margin: 0 0 6px;
    padding: 0;
    font-size: var(--text-sm);
  }

  li {
    display: flex;
    gap: 8px;
    padding: 2px 0;
  }

  .bad {
    color: var(--danger);
  }

  .muted {
    color: var(--fg-muted);
  }

  pre {
    margin: 0;
    padding: 8px 10px;
    border-radius: var(--control-radius);
    background: var(--content-bg);
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    white-space: pre-wrap;
    word-break: break-all;
  }
</style>
