<script lang="ts">
  import type { ClonePlan } from '../../lib/api/git'
  import { toUserError } from '../../lib/api/errors'
  import { t } from '../../lib/i18n'
  import Dialog from '../../lib/ui/Dialog.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import { openSelected, openTerminalAt } from '../../lib/wm/intents.svelte'
  import { backend } from './backend'

  interface Props {
    serverID: string
    /** clone, init a folder, or set a remote of repo; closed while null. */
    mode: 'clone' | 'init' | 'remote' | null
    /** The selected repository, for remote. */
    repo?: string
    onclose: () => void
  }

  let { serverID, mode, repo = '', onclose }: Props = $props()
  const g = t.git.clone

  let url = $state('')
  let dir = $state('')
  let branch = $state('')
  let remote = $state('origin')
  let info = $state.raw<ClonePlan | null>(null)
  let command = $state('')
  let error = $state('')
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
    error = ''
  })

  // Reads the URL as it is typed, to suggest /var/www/<name>.
  async function readURL() {
    error = ''
    if (!url.trim()) return void (info = null)
    try {
      info = await backend.planClone({ url, dir: '', branch: '' })
      if (!dirTouched && mode === 'clone') dir = `/var/www/${info.name}`
    } catch {
      info = null
    }
  }

  async function preview() {
    error = ''
    try {
      if (mode === 'clone') command = (await backend.planClone({ url, dir, branch })).command
      else if (mode === 'init') command = await backend.initCommand(dir, remote, url.trim())
      else command = await backend.remoteCommand(dir, remote, url)
    } catch (e) {
      error = toUserError(e).message
      command = ''
    }
  }

  function open() {
    openTerminalAt(serverID, undefined, command)
    onclose()
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
  {#if info?.ssh && mode === 'clone'}
    <div class="note">
      <p>{info.pinned ? g.sshPinned(info.host) : g.sshUnpinned(info.host)}</p>
      <button
        type="button"
        class="btn"
        onclick={() =>
          openSelected('keys', serverID, 'server', {
            title: t.apps.keys,
            size: { w: 900, h: 580 },
          })}
      >
        <Icon name="key" size={13} />
        {t.git.deployKey}
      </button>
    </div>
  {/if}
  {#if mode !== 'remote'}
    <label class="field">
      <span>{g.dir}</span>
      <input
        class="input"
        type="text"
        bind:value={dir}
        oninput={() => ((dirTouched = true), (command = ''))}
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
        oninput={() => (command = '')}
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
  {#if command}
    <p>{g.commandIntro}</p>
    <pre class="command">{command}</pre>
  {/if}
  {#if error}<p class="error" role="alert">{error}</p>{/if}

  {#snippet footer()}
    <span class="spacer"></span>
    <button type="button" class="btn" onclick={onclose}>{t.common.cancel}</button>
    {#if command}
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
