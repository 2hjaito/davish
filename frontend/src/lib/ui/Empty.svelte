<script lang="ts">
  import Icon from './Icon.svelte'
  import type { IconName } from './icons'

  interface Props {
    icon: IconName
    title: string
    hint?: string
  }

  let { icon, title, hint = '' }: Props = $props()
</script>

<div class="empty">
  <Icon name={icon} size={28} strokeWidth={1.6} />
  <strong>{title}</strong>
  {#if hint}<span>{hint}</span>{/if}
</div>

<style>
  .empty {
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

  strong {
    font-size: var(--text-md);
    color: var(--fg-muted);
  }

  span {
    max-width: 340px;
    font-size: var(--text-sm);
  }
</style>
