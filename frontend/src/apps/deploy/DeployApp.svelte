<script lang="ts">
  import { untrack } from 'svelte'

  import { type DeployPlan, type DeploySpec, deployPlan } from '../../lib/api/deploy'
  import { toUserError } from '../../lib/api/errors'
  import { hasRuntime } from '../../lib/api/runtime'
  import { t } from '../../lib/i18n'
  import { servers } from '../../lib/stores/servers.svelte'
  import Empty from '../../lib/ui/Empty.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import { openTerminalAt } from '../../lib/wm/intents.svelte'
  import { backend as nginx } from '../nginx/backend'
  import type { AppProps } from '../types'
  import { mockPlan } from './mock'

  let { win }: AppProps = $props()
  const w = t.deploy

  const server = $derived(servers.list.find((s) => s.id === win.serverID))
  const connected = $derived(server?.state === 'connected')

  type Page = 'source' | 'run' | 'domain' | 'review' | 'done'
  const PAGES: readonly Page[] = ['source', 'run', 'domain', 'review', 'done']
  let page = $state<Page>('source')

  let source = $state<DeploySpec['source']>('git')
  let repoURL = $state('')
  let branch = $state('')
  let image = $state('')
  let dir = $state('')
  let run = $state<DeploySpec['run']>('systemd')
  let port = $state(3000)
  let containerPort = $state(80)
  let name = $state('')
  let execStart = $state('/usr/bin/npm start')
  let user = $state('')
  let env = $state('')
  let envFile = $state('')
  let domains = $state('')
  let ssl = $state(true)
  let redirect = $state(true)
  let confD = $state(false)
  let dnsWarn = $state('')
  let plan = $state.raw<DeployPlan | null>(null)
  let error = $state('')

  // Where nginx keeps sites on this server.
  $effect(() => {
    if (connected)
      untrack(() => void nginx.siteOptions(win.serverID).then((o) => (confD = o.confD)))
  })

  const repoName = $derived(
    (repoURL.trim().split(/[/:]/).pop() ?? '').replace(/\.git$/, '').toLowerCase(),
  )
  const appDir = $derived(dir.trim() || (repoName ? `/var/www/${repoName}` : ''))
  const appName = $derived(
    name.trim() ||
      (source === 'image' ? (image.split(/[/:]/).at(-2) ?? '') : appDir.split('/').pop() || ''),
  )
  const runs = $derived<DeploySpec['run'][]>(
    source === 'image' ? ['container'] : ['systemd', 'compose', 'static'],
  )
  $effect(() => {
    if (!runs.includes(run)) run = runs[0] ?? 'systemd'
  })

  const spec = $derived<DeploySpec>({
    source,
    repoURL: repoURL.trim(),
    branch: branch.trim(),
    image: image.trim(),
    dir: appDir,
    run,
    port,
    containerPort,
    name: appName,
    unit: {
      name: appName,
      execStart,
      user: user.trim(),
      workingDir: appDir,
      env: env.split('\n'),
      envFile: envFile.trim(),
      restart: 'always',
    },
    domains: domains.split(/[\s,]+/).filter(Boolean),
    confD,
    ssl,
    redirect,
  })

  async function next() {
    error = ''
    const i = PAGES.indexOf(page)
    if (page === 'domain') {
      try {
        plan = hasRuntime ? await deployPlan(spec) : await mockPlan(spec)
      } catch (e) {
        error = toUserError(e).message
        return
      }
      dnsWarn = ''
      if (ssl && spec.domains.length > 0) {
        void nginx.planSSL(win.serverID, spec.domains, redirect).then((s) => {
          const off = s.dns.filter((d) => !d.ok).map((d) => d.name)
          dnsWarn = off.length ? w.dnsWarn(off.join(', ')) : ''
        })
      }
    }
    page = PAGES[i + 1] ?? page
  }

  function back() {
    error = ''
    page = PAGES[Math.max(0, PAGES.indexOf(page) - 1)] ?? 'source'
  }

  function start() {
    if (!plan) return
    openTerminalAt(win.serverID, undefined, plan.line)
    page = 'done'
  }

  function uninstall() {
    if (plan) openTerminalAt(win.serverID, undefined, plan.uninstall)
  }
</script>

{#if !connected}
  <Empty icon="server" title={w.offline} />
{:else}
  <div class="wizard">
    <ol class="steps">
      {#each PAGES as p, i (p)}
        <li class:on={p === page} class:past={PAGES.indexOf(page) > i}>{w.pages[p]}</li>
      {/each}
    </ol>

    <div class="body">
      {#if page === 'source'}
        <div class="choices">
          {#each ['git', 'image', 'folder'] as const as k (k)}
            <label class="choice" class:on={source === k}>
              <input type="radio" bind:group={source} value={k} />
              <strong>{w.sources[k]}</strong>
              <small>{w.sourceHints[k]}</small>
            </label>
          {/each}
        </div>
        {#if source === 'git'}
          <label class="field"
            ><span>{w.repoURL}</span><input
              class="input"
              bind:value={repoURL}
              placeholder="https://github.com/owner/app.git"
              spellcheck="false"
            /></label
          >
          <div class="row">
            <label class="field"
              ><span>{w.dir}</span><input
                class="input"
                bind:value={dir}
                placeholder={appDir || '/var/www/app'}
                spellcheck="false"
              /></label
            >
            <label class="field"
              ><span>{w.branch}</span><input
                class="input"
                bind:value={branch}
                placeholder={w.branchDefault}
                spellcheck="false"
              /></label
            >
          </div>
        {:else if source === 'image'}
          <label class="field"
            ><span>{w.image}</span><input
              class="input"
              bind:value={image}
              placeholder="ghost:5"
              spellcheck="false"
            /></label
          >
        {:else}
          <label class="field"
            ><span>{w.dir}</span><input
              class="input"
              bind:value={dir}
              placeholder="/var/www/app"
              spellcheck="false"
            /></label
          >
        {/if}
      {:else if page === 'run'}
        <div class="choices">
          {#each runs as k (k)}
            <label class="choice" class:on={run === k}>
              <input type="radio" bind:group={run} value={k} />
              <strong>{w.runs[k]}</strong>
              <small>{w.runHints[k]}</small>
            </label>
          {/each}
        </div>
        {#if run !== 'static'}
          <div class="row">
            <label class="field"
              ><span>{w.port}</span><input class="input" type="number" bind:value={port} /></label
            >
            {#if run === 'container'}
              <label class="field"
                ><span>{w.containerPort}</span><input
                  class="input"
                  type="number"
                  bind:value={containerPort}
                /></label
              >
            {/if}
            {#if run === 'systemd' || run === 'container'}
              <label class="field"
                ><span>{w.name}</span><input
                  class="input"
                  bind:value={name}
                  placeholder={appName}
                  spellcheck="false"
                /></label
              >
            {/if}
          </div>
        {/if}
        {#if run === 'systemd'}
          <label class="field"
            ><span>ExecStart</span><input
              class="input mono"
              bind:value={execStart}
              spellcheck="false"
            /><small>{w.execHint}</small></label
          >
          <div class="row">
            <label class="field"
              ><span>{w.user}</span><input
                class="input"
                bind:value={user}
                placeholder="www-data"
                spellcheck="false"
              /></label
            >
            <label class="field"
              ><span>{w.envFile}</span><input
                class="input"
                bind:value={envFile}
                placeholder="{appDir}/.env"
                spellcheck="false"
              /></label
            >
          </div>
          <label class="field"
            ><span>{w.env}</span><textarea
              class="input mono"
              rows="3"
              bind:value={env}
              placeholder="PORT={port}&#10;NODE_ENV=production"
              spellcheck="false"></textarea></label
          >
        {/if}
      {:else if page === 'domain'}
        <label class="field"
          ><span>{w.domains}</span><input
            class="input"
            bind:value={domains}
            placeholder="app.example.com"
            spellcheck="false"
          /><small>{w.domainsHint}</small></label
        >
        <label class="check"><input type="checkbox" bind:checked={ssl} /> {w.ssl}</label>
        {#if ssl}
          <label class="check"><input type="checkbox" bind:checked={redirect} /> {w.redirect}</label
          >
        {/if}
      {:else if page === 'review' && plan}
        <p class="muted">{w.reviewBody}</p>
        {#if dnsWarn}<p class="warn"><Icon name="warning" size={14} /> {dnsWarn}</p>{/if}
        {#each plan.steps as s, i (i)}
          <section class="step">
            <h4>{i + 1}. {w.stepTitles[s.title as keyof typeof w.stepTitles] ?? s.title}</h4>
            {#if s.file}
              <details>
                <summary>{w.file}</summary>
                <pre class="code">{s.file}</pre>
              </details>
            {/if}
            <pre class="code">{s.command}</pre>
          </section>
        {/each}
      {:else if page === 'done' && plan}
        <p>{w.doneBody}</p>
        <ul class="created">
          {#each plan.created as c (c)}<li><code>{c}</code></li>{/each}
        </ul>
        <h4>{w.uninstallTitle}</h4>
        <p class="muted">{w.uninstallBody}</p>
        <pre class="code">{plan.uninstall}</pre>
      {/if}
      {#if error}<p class="error" role="alert">{error}</p>{/if}
    </div>

    <footer>
      {#if page !== 'source' && page !== 'done'}
        <button type="button" class="btn" onclick={back}>{w.back}</button>
      {/if}
      <span class="spacer"></span>
      {#if page === 'review'}
        <button type="button" class="btn btn-primary" onclick={start}>{w.start}</button>
      {:else if page === 'done'}
        <button type="button" class="btn" onclick={() => (page = 'review')}>{w.back}</button>
        <button type="button" class="btn btn-danger" onclick={uninstall}>{w.uninstall}</button>
      {:else}
        <button
          type="button"
          class="btn btn-primary"
          disabled={page === 'source' && !(source === 'image' ? image.trim() : appDir)}
          onclick={() => void next()}>{w.next}</button
        >
      {/if}
    </footer>
  </div>
{/if}

<style>
  .wizard {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
  }

  .steps {
    display: flex;
    gap: 6px;
    margin: 0;
    padding: 12px 16px;
    list-style: none;
    border-bottom: 1px solid var(--border);
    font-size: var(--text-sm);
  }

  .steps li {
    padding: 3px 10px;
    border-radius: 999px;
    color: var(--fg-muted);
  }

  .steps li.past {
    color: var(--fg);
  }

  .steps li.on {
    background: var(--accent);
    color: var(--fg-on-accent);
  }

  .body {
    flex: 1;
    min-height: 0;
    padding: 16px;
    overflow: auto;
  }

  footer {
    display: flex;
    gap: 8px;
    padding: 12px 16px;
    border-top: 1px solid var(--border);
  }

  .spacer {
    flex: 1;
  }

  .choices {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 8px;
    margin: 0 0 14px;
  }

  .choice {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 10px;
    border: 1px solid var(--border);
    border-radius: var(--control-radius);
    cursor: pointer;
  }

  .choice.on {
    border-color: var(--accent);
    background: var(--fill-soft);
  }

  .choice input {
    position: absolute;
    opacity: 0;
    pointer-events: none;
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

  .check {
    display: flex;
    gap: 8px;
    align-items: center;
    margin: 0 0 10px;
  }

  small,
  .muted {
    color: var(--fg-muted);
    font-size: var(--text-xs);
  }

  .mono {
    font-family: var(--font-mono);
  }

  p {
    margin: 0 0 10px;
    line-height: 1.45;
  }

  .warn {
    display: flex;
    gap: 6px;
    color: var(--warn);
  }

  .step h4,
  h4 {
    margin: 12px 0 6px;
  }

  .code {
    max-height: 200px;
    margin: 0 0 6px;
    padding: 8px 10px;
    overflow: auto;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    font-family: var(--font-mono);
    font-size: var(--text-sm);
    white-space: pre-wrap;
    word-break: break-all;
    user-select: text;
  }

  .created {
    margin: 0 0 10px;
    padding-left: 18px;
  }

  .error {
    color: var(--danger);
  }
</style>
