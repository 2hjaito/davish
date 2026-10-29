<script lang="ts">
  import { t } from '../i18n'
  import { levelIcon, notifications } from '../stores/notifications.svelte'
  import Icon from '../ui/Icon.svelte'
  import Popover from './Popover.svelte'

  let open = $state(false)
</script>

<div class="anchor">
  <button
    class="mbtn bell"
    aria-label={t.menubar.notifications(notifications.unread)}
    aria-expanded={open}
    onclick={() => (open = !open)}
  >
    <Icon name="bell" size={15} />
    {#if notifications.unread > 0}<span class="badge">{notifications.unread}</span>{/if}
  </button>

  <Popover
    {open}
    onclose={() => (open = false)}
    label={t.notifications.title}
    width={300}
    align="right"
  >
    <div class="header">{t.notifications.title}</div>
    {#each notifications.list as n (n.id)}
      <div class="item">
        <span class="icon {n.level}"><Icon name={levelIcon(n.level)} size={14} /></span>
        <span class="text">
          <strong>{n.title}</strong>
          <span>{n.body}</span>
          {#if n.action}
            <button class="action" onclick={() => n.action?.run()}>{n.action.label}</button>
          {/if}
        </span>
        <button
          class="dismiss"
          aria-label={t.window.close}
          onclick={() => notifications.dismiss(n.id)}
        >
          <Icon name="close" size={12} />
        </button>
      </div>
    {:else}
      <p class="empty">{t.notifications.empty}</p>
    {/each}
  </Popover>
</div>

<style>
  .anchor {
    position: relative;
  }

  .bell {
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .badge {
    background: var(--danger);
    color: var(--fg-on-accent);
    font-size: 10px;
    font-weight: 700;
    border-radius: 8px;
    padding: 0 5px;
    line-height: 15px;
  }

  .header {
    padding: 6px 10px 4px;
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--fg-muted);
    letter-spacing: 0.04em;
    text-transform: uppercase;
  }

  .item {
    display: flex;
    gap: 10px;
    align-items: flex-start;
    padding: 8px 10px;
    border-radius: var(--item-radius);
  }

  .icon {
    color: var(--fg-muted);
    padding-top: 1px;
  }
  .icon.info {
    color: var(--accent);
  }
  .icon.warn {
    color: var(--warn);
  }
  .icon.error {
    color: var(--danger);
  }

  .text {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
    font-size: var(--text-sm);
    overflow-wrap: anywhere;
  }

  .text span {
    color: var(--fg-muted);
  }

  .action {
    align-self: flex-start;
    margin-top: 2px;
    padding: 0;
    border: 0;
    background: none;
    color: var(--accent);
    font-weight: 600;
  }

  .action:hover {
    text-decoration: underline;
  }

  .dismiss {
    border: 0;
    background: none;
    padding: 2px;
    color: var(--fg-subtle);
    border-radius: 4px;
  }

  .dismiss:hover {
    background: var(--fill-soft);
  }

  .empty {
    margin: 0;
    padding: 12px 10px;
    color: var(--fg-subtle);
  }
</style>
