<script lang="ts">
  import { avatarColor, avatarUrls, initials, markFailed } from './avatar'

  // An author's initials, with their picture over them once it loads.
  interface Props {
    name: string
    email?: string
  }

  let { name, email = '' }: Props = $props()

  let urls = $state<string[]>([])
  let loaded = $state(false)

  $effect(() => {
    const want = email
    urls = []
    loaded = false
    if (!want) return
    let live = true
    void avatarUrls(want).then((list) => {
      if (live) urls = list
    })
    return () => {
      live = false
    }
  })

  function onerror() {
    const bad = urls[0]
    if (bad) markFailed(bad)
    urls = urls.slice(1)
    loaded = false
  }
</script>

<span class="avatar" style:--avatar="var(--graph-{avatarColor(name) + 1})" aria-hidden="true">
  {#if !loaded}{initials(name)}{/if}
  {#if urls[0]}
    {#key urls[0]}
      <img
        src={urls[0]}
        alt=""
        loading="lazy"
        referrerpolicy="no-referrer"
        class:loaded
        onload={() => (loaded = true)}
        {onerror}
      />
    {/key}
  {/if}
</span>

<style>
  .avatar {
    position: relative;
    display: inline-grid;
    place-items: center;
    flex-shrink: 0;
    width: 20px;
    height: 20px;
    border-radius: 50%;
    overflow: hidden;
    background: color-mix(in srgb, var(--avatar) 20%, transparent);
    color: var(--avatar);
    font-size: 9px;
    font-weight: 700;
    letter-spacing: -0.02em;
  }

  img {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    object-fit: cover;
    opacity: 0;
  }

  img.loaded {
    opacity: 1;
  }

  :global(.scroller:focus-within .selected) .avatar {
    background: var(--fg-on-accent);
    color: var(--accent);
  }
</style>
