<script lang="ts">
  import { t } from '../i18n/vi'
  import { fadeOut, popIn } from '../motion'
  import { levelIcon, notifications } from '../stores/notifications.svelte'
  import Icon from '../ui/Icon.svelte'
</script>

<!-- New notifications slide in under the menu bar for a few seconds, as on
     macOS; the bell keeps them afterwards. -->
<section class="banners" aria-label={t.notifications.title} aria-live="polite">
  {#each notifications.banners as n (n.id)}
    <div
      class="banner glass glass-strong"
      role="status"
      onpointerenter={() => notifications.holdBanner(n.id)}
      onpointerleave={() => notifications.releaseBanner(n.id)}
      in:popIn
      out:fadeOut
    >
      <span class="icon {n.level}"><Icon name={levelIcon(n.level)} size={16} /></span>
      <span class="text">
        <strong>{n.title}</strong>
        {#if n.body}<span class="body">{n.body}</span>{/if}
        {#if n.action}
          <button
            class="action"
            onclick={() => {
              n.action?.run()
              notifications.hideBanner(n.id)
            }}>{n.action.label}</button
          >
        {/if}
      </span>
      <button
        class="close"
        aria-label={t.window.close}
        onclick={() => notifications.hideBanner(n.id)}
      >
        <Icon name="close" size={11} strokeWidth={2.4} />
      </button>
    </div>
  {/each}
</section>

<style>
  .banners {
    position: fixed;
    top: calc(var(--menubar-h) + 8px);
    right: 10px;
    z-index: 950;
    width: 340px;
    display: flex;
    flex-direction: column;
    gap: 8px;
    pointer-events: none;
  }

  .banner {
    position: relative;
    display: flex;
    gap: 10px;
    align-items: flex-start;
    padding: 11px 14px;
    border-radius: var(--menu-radius);
    pointer-events: auto;
    font-size: var(--text-sm);
  }

  .icon {
    padding-top: 1px;
    color: var(--fg-muted);
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
    overflow-wrap: anywhere;
  }

  .body {
    color: var(--fg-muted);
  }

  .action {
    align-self: flex-start;
    margin-top: 4px;
    padding: 3px 10px;
    border: 0;
    border-radius: var(--control-radius);
    background: var(--accent-soft);
    color: var(--accent);
    font-weight: 600;
    transition: background var(--dur-fast) var(--ease-out);
  }

  .action:hover {
    background: var(--row-hover);
  }

  /* Like macOS: the close button sits on the corner and shows on hover. */
  .close {
    position: absolute;
    top: -6px;
    left: -6px;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 20px;
    height: 20px;
    padding: 0;
    border: 1px solid var(--glass-border);
    border-radius: 50%;
    background: var(--glass-bg-strong);
    color: var(--fg-muted);
    opacity: 0;
    transition: opacity var(--dur-fast) var(--ease-out);
  }

  .banner:hover .close,
  .close:focus-visible {
    opacity: 1;
  }
</style>
