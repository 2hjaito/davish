<script lang="ts">
  import type { Snippet } from 'svelte'

  import Icon from '../../lib/ui/Icon.svelte'
  import type { IconName } from '../../lib/ui/icons'

  interface Props {
    title: string
    icon: IconName
    /** Shown at the right of the header: a legend, a badge. */
    aside?: Snippet
    /** Makes the title a link to another section. */
    onopen?: () => void
    children: Snippet
  }

  let { title, icon, aside, onopen, children }: Props = $props()
</script>

<section class="card">
  <header>
    {#if onopen}
      <button class="title link" onclick={onopen}>
        <Icon name={icon} size={15} />
        <h3>{title}</h3>
        <Icon name="chevronRight" size={13} strokeWidth={2.2} />
      </button>
    {:else}
      <span class="title">
        <Icon name={icon} size={15} />
        <h3>{title}</h3>
      </span>
    {/if}
    {#if aside}<div class="aside">{@render aside()}</div>{/if}
  </header>
  {@render children()}
</section>

<style>
  .card {
    display: flex;
    flex-direction: column;
    gap: 12px;
    min-width: 0;
    padding: 12px 14px 14px;
    border-radius: var(--card-radius);
    background: var(--card-bg);
    border: 1px solid var(--divider-soft);
  }

  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    min-height: 22px;
  }

  .title {
    display: flex;
    align-items: center;
    gap: 7px;
    color: var(--accent);
  }

  .link {
    margin: -3px -6px;
    padding: 3px 6px;
    border: 0;
    border-radius: var(--control-radius);
    background: none;
    transition: background var(--dur-fast) var(--ease-out);
  }

  .link:hover {
    background: var(--fill-soft);
  }

  .link :global(svg:last-child) {
    color: var(--fg-subtle);
  }

  h3 {
    margin: 0;
    font-size: var(--text-md);
    font-weight: 600;
    color: var(--fg);
  }

  .aside {
    display: flex;
    align-items: center;
    gap: 12px;
    font-size: var(--text-xs);
    color: var(--fg-muted);
  }
</style>
