<script lang="ts">
  import { profile } from '../stores/profile.svelte'
  import Icon from './Icon.svelte'

  // The user's picture, or their initials, or a plain figure without a name.
  interface Props {
    size?: number
  }

  let { size = 36 }: Props = $props()

  const letters = $derived(
    profile.name
      .split(/\s+/)
      .filter(Boolean)
      .map((w) => [...w][0] ?? '')
      .filter((_, i, all) => i === 0 || i === all.length - 1)
      .join('')
      .toUpperCase(),
  )
</script>

<span class="avatar" style:--size="{size}px" aria-hidden="true">
  {#if profile.avatar}
    <img src={profile.avatar} alt="" />
  {:else if letters}
    {letters}
  {:else}
    <Icon name="user" size={Math.round(size * 0.55)} />
  {/if}
</span>

<style>
  .avatar {
    flex-shrink: 0;
    display: inline-grid;
    place-items: center;
    width: var(--size);
    height: var(--size);
    border-radius: 50%;
    overflow: hidden;
    background: linear-gradient(180deg, #a5abb8, #6f7584);
    color: #fff;
    font-size: calc(var(--size) * 0.4);
    font-weight: 600;
    letter-spacing: 0.01em;
  }

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
</style>
