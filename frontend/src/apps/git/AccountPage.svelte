<script lang="ts">
  import type { HostLogin, Identity } from '../../lib/api/git'
  import { t } from '../../lib/i18n'
  import Empty from '../../lib/ui/Empty.svelte'
  import Icon from '../../lib/ui/Icon.svelte'
  import { openSelected } from '../../lib/wm/intents.svelte'
  import { accountName, accountPicture } from './account'
  import Avatar from './Avatar.svelte'

  // Who the server works as for git: the SSH user, git's own name and
  // email, and the code host accounts its keys log in to. Read only.
  interface Props {
    serverID: string
    identity: Identity | null
    error: string
  }

  let { serverID, identity, error }: Props = $props()
  const a = t.git.account

  function openKeys() {
    openSelected('keys', serverID, 'server', { title: t.apps.keys, size: { w: 900, h: 580 } })
  }

  function loginText(h: HostLogin): string {
    if (h.state === 'ok') return h.deploy ? a.loginDeploy(h.account) : a.loginOk(h.account)
    return h.state === 'denied' ? a.loginDenied : a.loginUnreachable
  }
</script>

{#if error && !identity}
  <Empty icon="warning" title={a.failed} hint={error} />
{:else if !identity}
  <Empty icon="gitBranch" title={a.loading} />
{:else}
  <div class="page">
    <header>
      <Avatar
        name={accountName(identity) || '?'}
        email={identity.gitEmail}
        src={accountPicture(identity)}
        size={56}
      />
      <span class="who">
        <strong>{accountName(identity)}</strong>
        <small>{identity.user}@{identity.host}</small>
      </span>
    </header>

    <h3>{a.server}</h3>
    <div class="card">
      <div class="row"><span>{a.sshUser}</span><code>{identity.user}</code></div>
      <div class="row"><span>{a.home}</span><code>{identity.home}</code></div>
      <div class="row"><span>{a.hostname}</span><code>{identity.host}</code></div>
    </div>

    <h3>Git</h3>
    <div class="card">
      <div class="row">
        <span>{a.gitName}</span>
        {#if identity.gitName}<code>{identity.gitName}</code>{:else}<em>{a.notSet}</em>{/if}
      </div>
      <div class="row">
        <span>{a.gitEmail}</span>
        {#if identity.gitEmail}<code>{identity.gitEmail}</code>{:else}<em>{a.notSet}</em>{/if}
      </div>
      <div class="row">
        <span>{a.helper}</span>
        {#if identity.helper}<code>{identity.helper}</code>{:else}<em>{a.noHelper}</em>{/if}
      </div>
    </div>
    <p class="hint">{a.gitHint}</p>

    <h3>{a.hosts}</h3>
    <div class="card">
      {#each identity.hosts as h (h.host)}
        <div class="row">
          <span>{h.host}</span>
          <span class="state {h.state}">
            <Icon name={h.state === 'ok' ? 'check' : 'warning'} size={13} />
            {loginText(h)}
          </span>
        </div>
      {/each}
    </div>
    {#if !identity.hosts.some((h) => h.state === 'ok')}
      <div class="note">
        <p>{a.noLogin}</p>
        <button type="button" class="btn" onclick={openKeys}>
          <Icon name="key" size={13} />
          {t.git.deployKey}
        </button>
      </div>
    {/if}

    <h3>{a.keys}</h3>
    <div class="card">
      {#each identity.keys as k (k.fingerprint)}
        <div class="row">
          <span class="key">
            <span>{k.type} · {k.bits}</span>
            {#if k.comment}<small>{k.comment}</small>{/if}
          </span>
          <code class="fp">{k.fingerprint}</code>
        </div>
      {:else}
        <div class="row"><em>{a.noKeys}</em></div>
      {/each}
    </div>
    <div class="actions">
      <button type="button" class="btn" onclick={openKeys}>
        <Icon name="key" size={13} />
        {a.openKeys}
      </button>
    </div>

    <p class="hint">{a.readOnly}</p>
  </div>
{/if}

<style>
  .page {
    max-width: 640px;
    margin: 0 auto;
    padding: 20px 24px 28px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  header {
    display: flex;
    align-items: center;
    gap: 14px;
    margin-bottom: 8px;
  }

  .who {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }

  .who strong {
    font-size: var(--text-lg);
    font-weight: 600;
  }

  h3 {
    margin: 10px 0 0;
    font-size: var(--text-sm);
    font-weight: 600;
  }

  .card {
    display: flex;
    flex-direction: column;
    padding: 0 14px;
    border-radius: 10px;
    background: var(--fill-soft);
  }

  .row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    min-height: 38px;
    padding: 8px 0;
  }

  .row + .row {
    border-top: 1px solid var(--divider-soft);
  }

  code {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-family: var(--font-mono);
    font-size: var(--text-sm);
  }

  .fp {
    color: var(--fg-muted);
    font-size: var(--text-xs);
  }

  em {
    font-style: normal;
    color: var(--fg-muted);
  }

  small {
    color: var(--fg-muted);
    font-size: var(--text-xs);
  }

  .key {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }

  .state {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    color: var(--fg-muted);
  }

  .state.ok {
    color: var(--ok);
  }

  .state.denied {
    color: var(--warn);
  }

  .note {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 14px;
    border-radius: 10px;
    background: var(--fill-soft);
  }

  .note p {
    flex: 1;
    margin: 0;
  }

  .actions {
    display: flex;
  }

  .hint {
    margin: 0;
    color: var(--fg-muted);
    font-size: var(--text-xs);
  }
</style>
