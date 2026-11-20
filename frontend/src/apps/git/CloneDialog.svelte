<script lang="ts">
  import type { ClonePlan, Identity } from '../../lib/api/git'
  import { toUserError } from '../../lib/api/errors'
  import { t } from '../../lib/i18n'
  import Dialog from '../../lib/ui/Dialog.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import { openSelected, openTerminalAt } from '../../lib/wm/intents.svelte'
  import { sshReadiness } from './account'
  import { backend } from './backend'

  interface Props {
    serverID: string
    /** clone, init a folder, or set a remote of repo; closed while null. */
    mode: 'clone' | 'init' | 'remote' | null
    /** The selected repository, for remote. */
    repo?: string
    /** Who the server works as, to tell whether an SSH clone can run here. */
    identity?: Identity | null
    /** Code hosts still being asked which account the keys sign in as. */
    pending?: string[]
    /** A clone finished in dir. */
    oncloned?: (dir: string) => void
    onclose: () => void
  }

  let {
    serverID,
    mode,
    repo = '',
    identity = null,
    pending = [],
    oncloned,
    onclose,
  }: Props = $props()
  const g = t.git.clone

  let url = $state('')
  let dir = $state('')
  let branch = $state('')
  let remote = $state('origin')
  let info = $state.raw<ClonePlan | null>(null)
  let command = $state('')
  let error = $state('')
  /** What git printed when a clone failed. */
  let output = $state('')
  /** The remote wants a password or token: only a terminal can ask. */
  let needsTerminal = $state(false)
  let cloning = $state(false)
  let dirTouched = false

  $effect(() => {
    if (!mode) return
    url = ''
    branch = ''
    remote = 'origin'
    dir = mode === 'remote' ? repo : ''
    dirTouched = mode === 'remote'
    info = null
    command = ''
    reset()
  })

  function reset() {
    error = ''
    output = ''
    needsTerminal = false
  }

  // What the server's keys get from the URL's host over SSH, when asked.
  const login = $derived(info?.ssh ? sshReadiness(identity, info.host) : null)

  // Reads the URL as it is typed, to suggest /var/www/<name>.
  async function readURL() {
    reset()
    if (!url.trim()) return void (info = null)
    try {
      info = await backend.planClone({ url, dir: '', branch: '' })
      if (!dirTouched && mode === 'clone') dir = `/var/www/${info.name}`
    } catch {
      info = null
    }
  }

  async function preview() {
    reset()
    try {
      if (mode === 'clone') command = (await backend.planClone({ url, dir, branch })).command
      else if (mode === 'init') command = await backend.initCommand(dir, remote, url.trim())
      else command = await backend.remoteCommand(dir, remote, url)
    } catch (e) {
      error = toUserError(e).message
      command = ''
    }
  }

  // A clone runs here; one that needs a login goes to a terminal instead.
  async function clone() {
    if (cloning) return
    reset()
    cloning = true
    try {
      const spec = { url, dir: dir.trim(), branch }
      const r = await backend.clone(serverID, spec)
      if (!r.error) {
        oncloned?.(spec.dir)
        onclose()
        return
      }
      error = toUserError({ code: r.error, message: r.output }).message
      output = r.output
      if (r.error === 'GIT_AUTH_FAILED') {
        needsTerminal = true
        command = (await backend.planClone(spec)).command
      }
    } catch (e) {
      error = toUserError(e).message
    } finally {
      cloning = false
    }
  }

  function open() {
    openTerminalAt(serverID, undefined, command)
    onclose()
  }

  function openKeys() {
    openSelected('keys', serverID, 'server', { title: t.apps.keys, size: { w: 900, h: 580 } })
  }
</script>

<Dialog open={mode !== null} title={mode ? g.title[mode] : ''} width={540} {onclose}>
  <label class="field">
    <span>{mode === 'init' ? g.urlOptional : g.url}</span>
    <!-- svelte-ignore a11y_autofocus -->
    <input
      class="input"
      type="text"
      bind:value={url}
      oninput={() => ((command = ''), void readURL())}
      placeholder="git@github.com:owner/app.git"
      autofocus
      spellcheck="false"
    />
  </label>
  {#if info && mode === 'clone'}
    {#if !info.ssh}
      <p class="status"><Icon name="info" size={13} /> {g.httpsNote}</p>
    {:else if pending.includes(info.host)}
      <p class="status">{g.checking(info.host)}</p>
    {:else if login?.state === 'ok'}
      <p class="status ok">
        <Icon name="check" size={13} />
        {login.deploy
          ? g.readyDeploy(login.host, login.account)
          : g.readyAs(login.host, login.account)}
      </p>
    {:else}
      <div class="note">
        <p>{login?.state === 'denied' ? g.sshDenied(info.host) : g.sshUnknown(info.host)}</p>
        <button type="button" class="btn" onclick={openKeys}>
          <Icon name="key" size={13} />
          {t.git.deployKey}
        </button>
      </div>
    {/if}
  {/if}
  {#if mode !== 'remote'}
    <label class="field">
      <span>{g.dir}</span>
      <input
        class="input"
        type="text"
        bind:value={dir}
        oninput={() => ((dirTouched = true), (command = ''), reset())}
        placeholder="/var/www/app"
        spellcheck="false"
      />
    </label>
  {/if}
  {#if mode === 'clone'}
    <label class="field">
      <span>{g.branch}</span>
      <input
        class="input"
        type="text"
        bind:value={branch}
        oninput={() => ((command = ''), reset())}
        placeholder={g.branchDefault}
        spellcheck="false"
      />
    </label>
  {:else}
    <label class="field">
      <span>{g.remote}</span>
      <input
        class="input"
        type="text"
        bind:value={remote}
        oninput={() => (command = '')}
        spellcheck="false"
      />
      {#if mode === 'remote'}<small>{g.remoteHint(repo)}</small>{/if}
    </label>
  {/if}
  {#if mode === 'clone'}
    {#if needsTerminal}
      <p>{g.needsTerminal}</p>
      <pre class="command">{command}</pre>
    {/if}
  {:else if command}
    <p>{g.commandIntro}</p>
    <pre class="command">{command}</pre>
  {/if}
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  {#if output && !needsTerminal}<pre class="command output">{output}</pre>{/if}

  {#snippet footer()}
    <span class="spacer"></span>
    <button type="button" class="btn" onclick={onclose}>{t.common.cancel}</button>
    {#if mode === 'clone'}
      {#if needsTerminal}
        <button type="button" class="btn btn-primary" onclick={open}>{g.open}</button>
      {:else}
        <button
          type="button"
          class="btn btn-primary"
          disabled={cloning || !dir.trim() || !url.trim()}
          onclick={() => void clone()}>{cloning ? g.cloning : g.run}</button
        >
      {/if}
    {:else if command}
      <button type="button" class="btn btn-primary" onclick={open}>{g.open}</button>
    {:else}
      <button
        type="button"
        class="btn btn-primary"
        disabled={!dir.trim() || (mode !== 'init' && !url.trim())}
        onclick={() => void preview()}>{g.next}</button
      >
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

  .note {
    display: flex;
    gap: 10px;
    align-items: center;
    margin: 0 0 12px;
    padding: 8px 10px;
    border-radius: var(--control-radius);
    background: var(--fill-soft);
    font-size: var(--text-sm);
  }

  .note p {
    flex: 1;
    margin: 0;
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
    -webkit-user-select: text;
    user-select: text;
  }

  .status {
    display: flex;
    gap: 6px;
    align-items: center;
    margin: -4px 0 12px;
    color: var(--fg-muted);
    font-size: var(--text-sm);
  }

  .status.ok {
    color: var(--ok);
  }

  .output {
    max-height: 140px;
    margin-top: 8px;
    overflow: auto;
    font-size: var(--text-xs);
  }

  .error {
    margin: 10px 0 0;
    color: var(--danger);
  }

  .spacer {
    flex: 1;
  }
</style>
