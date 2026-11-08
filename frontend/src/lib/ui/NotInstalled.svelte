<script lang="ts">
  import { toUserError } from '../api/errors'
  import { packagesSetup, type Setup, type SetupTool } from '../api/packages'
  import { t } from '../i18n'
  import { servers } from '../stores/servers.svelte'
  import Icon from './Icon.svelte'
  import type { IconName } from './icons'
  import NeedsRoot from './NeedsRoot.svelte'

  // What an app shows when the server lacks the program it drives: the
  // command that installs it for the server's distro, typed into a
  // terminal for the user to run (docs/SECURITY.md).
  interface Props {
    serverID: string
    tool: SetupTool
    /** The program's name, e.g. "Docker". */
    name: string
    icon?: IconName
    /** Looks again, after the user installed it. */
    onretry: () => void
    /** Inside a section that already names the program: no heading. */
    compact?: boolean
  }

  let { serverID, tool, name, icon = 'package', onretry, compact = false }: Props = $props()

  const n = t.install

  let setup = $state.raw<Setup | null>(null)
  let error = $state('')
  let reconnecting = $state(false)

  $effect(() => {
    const id = serverID
    const which = tool
    setup = null
    error = ''
    let stale = false
    packagesSetup(id, which).then(
      (s) => {
        if (!stale) setup = s
      },
      (e: unknown) => {
        if (!stale) error = toUserError(e).message
      },
    )
    return () => {
      stale = true
    }
  })

  // A new group only applies to a new SSH connection; reconnect alone
  // keeps the live one.
  async function relogin() {
    reconnecting = true
    try {
      await servers.disconnect(serverID)
      await servers.reconnect(serverID)
    } finally {
      reconnecting = false
    }
  }
</script>

<div class="missing" class:compact>
  {#if !compact}
    <Icon name={icon} size={28} strokeWidth={1.6} />
    <strong>{n.title(name)}</strong>
    <span class="hint">{n.hint}</span>
  {/if}

  <div class="setup">
    {#if error}
      <p class="note">{error}</p>
    {:else if !setup}
      <p class="note">{n.planning}</p>
    {:else if !setup.command}
      <p class="note">{n.unknownManager}</p>
    {:else}
      <NeedsRoot {serverID} command={setup.command} message={n.needsRoot} />
      {#if tool === 'docker'}
        <p class="note warn"><Icon name="warning" size={13} />{n.dockerGroup}</p>
      {/if}
      {#if setup.relogin}
        <p class="note">
          {n.relogin}
          <button type="button" class="link" disabled={reconnecting} onclick={() => void relogin()}>
            {n.reconnect}
          </button>
        </p>
      {/if}
    {/if}
  </div>

  <button type="button" class="btn" onclick={onretry}>{n.check}</button>
</div>

<style>
  .missing {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 6px;
    min-height: 200px;
    padding: 24px;
    text-align: center;
    color: var(--fg-subtle);
  }

  .compact {
    flex: none;
    align-items: flex-start;
    min-height: 0;
    padding: 0;
  }

  .compact .setup {
    margin-top: 0;
  }

  strong {
    font-size: var(--text-md);
    color: var(--fg-muted);
  }

  .hint {
    max-width: 380px;
    font-size: var(--text-sm);
  }

  .setup {
    width: min(460px, 100%);
    margin: 10px 0 6px;
    text-align: left;
    color: var(--fg);
  }

  .note {
    margin: 8px 0 0;
    font-size: var(--text-sm);
    line-height: 1.4;
    color: var(--fg-muted);
  }

  .warn {
    display: flex;
    gap: 6px;
  }

  .warn :global(svg) {
    flex-shrink: 0;
    margin-top: 2px;
    color: var(--warn);
  }

  .btn {
    color: var(--fg);
  }

  .link {
    padding: 0;
    border: 0;
    background: none;
    font: inherit;
    color: var(--accent);
    cursor: pointer;
  }

  .link:disabled {
    opacity: 0.5;
    cursor: default;
  }
</style>
