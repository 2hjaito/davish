<script lang="ts">
  import type { Snippet } from 'svelte'

  import Icon from '../ui/Icon.svelte'
  import type { IconName } from '../ui/icons'

  // The 52px square of a desktop icon: a picture, or a glyph on glass or a color.
  interface Props {
    icon: IconName
    color?: string
    /** A data URL; wins over icon and color. */
    image?: string
    running?: boolean
    children?: Snippet
  }

  let { icon, color, image, running = false, children }: Props = $props()
</script>

<span
  class="tile"
  class:glass={!image}
  class:tinted={!image && !!color}
  class:running
  style:background={image ? undefined : color}
>
  {#if image}
    <img src={image} alt="" draggable="false" />
  {:else}
    <Icon name={icon} size={24} />
  {/if}
  {@render children?.()}
</span>

<style>
  .tile {
    position: relative;
    display: grid;
    place-items: center;
    width: 52px;
    height: 52px;
    flex: none;
    border-radius: var(--dock-tile-radius);
  }

  .tinted {
    color: var(--fg-on-accent);
  }

  img {
    display: block;
    width: 100%;
    height: 100%;
    border-radius: inherit;
    box-shadow: var(--glass-shadow);
    object-fit: cover;
  }

  .running {
    box-shadow: 0 0 0 2px var(--accent);
    animation: pulse 1.6s ease-in-out infinite;
  }

  @keyframes pulse {
    50% {
      box-shadow: 0 0 0 2px transparent;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .running {
      animation: none;
    }
  }
</style>
