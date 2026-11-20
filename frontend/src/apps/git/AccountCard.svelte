<script lang="ts">
  import type { Identity } from '../../lib/api/git'
  import { t } from '../../lib/i18n'
  import { accountName, accountPicture, mainLogin } from './account'
  import Avatar from './Avatar.svelte'

  // The top of the Git sidebar, like the account in System Settings: who
  // the server works as; a click shows the details.
  interface Props {
    identity: Identity | null
    active: boolean
    onclick: () => void
  }

  let { identity, active, onclick }: Props = $props()

  const name = $derived(accountName(identity))
  const login = $derived(mainLogin(identity))
  const sub = $derived(
    !identity
      ? t.git.account.loading
      : login
        ? `${identity.user}@${identity.host} · ${login.host}`
        : `${identity.user}@${identity.host}`,
  )
</script>

<button
  type="button"
  class="card"
  class:active
  aria-current={active ? 'page' : undefined}
  title={t.git.account.title}
  {onclick}
>
  <Avatar
    name={name || '?'}
    email={identity?.gitEmail ?? ''}
    src={accountPicture(identity)}
    size={36}
  />
  <span class="text">
    <strong>{name || t.git.account.title}</strong>
    <small>{sub}</small>
  </span>
</button>

<style>
  .card {
    display: flex;
    align-items: center;
    gap: 9px;
    margin-bottom: 8px;
    padding: 6px 8px;
    text-align: left;
    border: 0;
    background: none;
    color: inherit;
    border-radius: var(--control-radius);
    transition: background var(--dur-fast) var(--ease-out);
  }

  .card:hover:not(.active) {
    background: var(--row-hover);
  }

  .card.active {
    background: var(--sidebar-active);
  }

  .text {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }

  strong,
  small {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  strong {
    font-weight: 600;
  }

  small {
    color: var(--fg-muted);
    font-size: var(--text-xs);
  }
</style>
