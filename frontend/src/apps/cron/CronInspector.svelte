<script lang="ts">
  import { untrack } from 'svelte'

  import type { CronJob, CronOp, CronRuns } from '../../lib/api/cron'
  import { toUserError } from '../../lib/api/errors'
  import { t } from '../../lib/i18n'
  import Icon from '../../lib/ui/Icon.svelte'
  import NeedsRoot from '../../lib/ui/NeedsRoot.svelte'
  import { sudoPrompt } from '../../lib/stores/sudoPrompt.svelte'
  import { openFileInEditor, openFilesAt } from '../../lib/wm/intents.svelte'
  import { backend } from './backend'
  import { describe } from './describe'
  import { commandPaths, folderOf } from './paths'
  import { formatRun } from './when'

  interface Props {
    serverID: string
    job: CronJob
    zone: string
    /** Starts a change; edit opens the job dialog. */
    onchange: (op: Exclude<CronOp, 'add'>, job: CronJob) => void
  }

  let { serverID, job, zone, onchange }: Props = $props()

  let runs = $state.raw<CronRuns | null>(null)
  let runsError = $state('')
  /** What the command mentions that exists on the server. */
  let paths = $state.raw<{ path: string; dir: boolean }[]>([])
  const id = $derived(job.id)
  const sourceLabel = $derived(
    job.source === 'user' ? t.cron.mine : job.source === 'root' ? t.cron.rootSource : job.source,
  )

  $effect(() => {
    void id
    untrack(() => {
      void loadRuns()
      void loadPaths()
    })
  })

  async function loadPaths() {
    const command = job.command
    paths = []
    try {
      // Another user's job, or root's, runs from that user's home, which is
      // unknown here.
      const home = job.user || job.source === 'root' ? '' : await backend.home(serverID)
      const found = await Promise.all(
        commandPaths(command, home).map(async (path) => {
          try {
            return { path, dir: await backend.isDir(serverID, path) }
          } catch {
            return null
          }
        }),
      )
      if (job.command === command) paths = found.filter((p) => p !== null)
    } catch {
      // Without links the job still shows; they are a convenience.
    }
  }

  async function loadRuns() {
    const command = job.command
    runs = null
    runsError = ''
    try {
      const r = await backend.runs(serverID, command)
      if (job.command === command) runs = r
    } catch (e) {
      if (job.command === command) runsError = toUserError(e).message
    }
  }
</script>

<section class="inspector" aria-label={describe(job.schedule)}>
  <header>
    <Icon name="clock" size={18} />
    <div class="heading">
      <h3>{describe(job.schedule)}</h3>
      <p class="raw">
        <code>{job.schedule.raw}</code>
        {#if !job.enabled}<span class="badge">{t.cron.off}</span>{/if}
      </p>
    </div>
  </header>

  {#if job.editable}
    <div class="actions">
      <button
        type="button"
        class="btn"
        onclick={() => onchange(job.enabled ? 'disable' : 'enable', job)}
      >
        {job.enabled ? t.cron.disable : t.cron.enable}
      </button>
      <button type="button" class="btn" onclick={() => onchange('edit', job)}>{t.cron.edit}</button>
      <button type="button" class="btn" onclick={() => onchange('delete', job)}
        >{t.cron.remove}</button
      >
    </div>
  {:else}
    <p class="hint">{t.cron.systemHint}</p>
    <div class="actions">
      <button
        type="button"
        class="btn"
        onclick={() => openFileInEditor(serverID, job.source, job.line)}
      >
        {t.cron.openFile}
      </button>
      <button type="button" class="btn" onclick={() => openFilesAt(serverID, folderOf(job.source))}>
        {t.cron.showFolder}
      </button>
    </div>
  {/if}

  <dl class="facts">
    <dt>{t.cron.command}</dt>
    <dd><code class="cmd">{job.command}</code></dd>
    {#if paths.length > 0}
      <dt>{t.cron.files}</dt>
      <dd>
        <ul class="paths">
          {#each paths as p (p.path)}
            <li>
              <code title={p.path}>{p.path}</code>
              {#if !p.dir}
                <button
                  type="button"
                  class="icon-btn"
                  aria-label={t.cron.editFile(p.path)}
                  title={t.cron.editFile(p.path)}
                  onclick={() => openFileInEditor(serverID, p.path)}
                >
                  <Icon name="editor" size={13} />
                </button>
              {/if}
              <button
                type="button"
                class="icon-btn"
                aria-label={t.cron.openFolder(p.path)}
                title={t.cron.openFolder(p.path)}
                onclick={() => openFilesAt(serverID, p.dir ? p.path : folderOf(p.path))}
              >
                <Icon name="folder" size={13} />
              </button>
            </li>
          {/each}
        </ul>
      </dd>
    {/if}
    <dt>{t.cron.source}</dt>
    <dd>
      {sourceLabel}{#if job.user}
        · {t.cron.asUser(job.user)}{/if}
    </dd>
    <dt>{t.cron.nextRuns}</dt>
    <dd>
      {#if !job.enabled}<span class="muted">{t.cron.off}</span>
      {:else if job.next.length === 0}<span class="muted"
          >{job.schedule.reboot ? t.cron.atBoot : t.cron.never}</span
        >
      {:else}
        <ul class="next" title={t.cron.zone(zone)}>
          {#each job.next as n (n)}<li>{formatRun(n, zone)}</li>{/each}
        </ul>
      {/if}
    </dd>
  </dl>

  <div class="runs-head">
    <h4>{t.cron.runs}</h4>
    <button
      type="button"
      class="icon-btn"
      aria-label={t.cron.refreshRuns}
      title={t.cron.refreshRuns}
      onclick={() => void loadRuns()}
    >
      <Icon name="refresh" size={13} strokeWidth={2.2} />
    </button>
  </div>
  {#if runsError}
    <p class="error" role="alert">{runsError}</p>
  {:else if !runs}
    <p class="muted small">{t.cron.runsLoading}</p>
  {:else if runs.restricted}
    <NeedsRoot {serverID} command={runs.sudo} message={t.cron.runsRestricted} />
    {#if runs.askSudo}
      <button
        type="button"
        class="btn sudo"
        onclick={async () => {
          if (runs && (await sudoPrompt.ask(serverID, runs.sudo))) void loadRuns()
        }}
      >
        <Icon name="key" size={13} />
        {t.sudo.journal}
      </button>
    {/if}
  {:else if runs.lines.length === 0}
    <p class="muted small">{t.cron.runsEmpty}</p>
  {:else}
    <!-- Server output is shown as text only (docs/SECURITY.md). -->
    <pre class="log">{runs.lines.join('\n')}</pre>
  {/if}
</section>

<style>
  .sudo {
    align-self: flex-start;
    margin-top: 6px;
  }

  .paths {
    display: flex;
    flex-direction: column;
    gap: 2px;
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .paths li {
    display: flex;
    align-items: center;
    gap: 4px;
    min-width: 0;
  }

  .paths code {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

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
    gap: 10px;
    align-items: flex-start;
  }

  header > :global(svg) {
    flex-shrink: 0;
    margin-top: 2px;
    color: var(--accent);
  }

  .heading {
    min-width: 0;
  }

  h3 {
    margin: 0;
    font-size: var(--text-lg);
    font-weight: 600;
  }

  .raw {
    display: flex;
    align-items: center;
    gap: 8px;
    margin: 3px 0 0;
  }

  code {
    font-family: var(--font-mono);
    font-size: var(--text-sm);
    user-select: text;
  }

  .raw code {
    color: var(--fg-muted);
  }

  .badge {
    padding: 0 7px;
    border-radius: 8px;
    background: var(--fill-soft);
    font-size: var(--text-xs);
    color: var(--fg-muted);
  }

  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .hint {
    margin: 0;
    font-size: var(--text-sm);
    color: var(--fg-muted);
    line-height: 1.4;
  }

  .facts {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: 6px 14px;
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

  .cmd {
    white-space: pre-wrap;
    word-break: break-all;
  }

  .next {
    margin: 0;
    padding: 0;
    list-style: none;
    font-variant-numeric: tabular-nums;
  }

  .muted {
    color: var(--fg-muted);
  }

  .small {
    margin: 0;
    font-size: var(--text-sm);
  }

  .runs-head {
    display: flex;
    align-items: center;
    margin-top: 4px;
  }

  h4 {
    flex: 1;
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

  .error {
    margin: 0;
    font-size: var(--text-sm);
    color: var(--danger);
  }

  .log {
    margin: 0;
    padding: 8px 10px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    line-height: 1.5;
    white-space: pre-wrap;
    word-break: break-word;
    user-select: text;
  }
</style>
