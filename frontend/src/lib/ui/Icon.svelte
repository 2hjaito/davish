<script lang="ts">
  import { icons, type IconName, type IconNode } from './icons'

  interface Props {
    name: IconName
    size?: number | string
    /** Overrides the pack's stroke width, e.g. 1.8 for dock tiles. */
    strokeWidth?: number
    /** Accessible label; omit for decorative icons. */
    title?: string
  }

  let { name, size = 16, strokeWidth, title }: Props = $props()

  const icon = $derived(icons[name])

  function withStroke(attrs: IconNode[1]): IconNode[1] {
    if (strokeWidth === undefined || !('stroke-width' in attrs)) return attrs
    return { ...attrs, 'stroke-width': strokeWidth }
  }
</script>

{#snippet node([tag, attrs, children]: IconNode)}
  <svelte:element this={tag} xmlns="http://www.w3.org/2000/svg" {...withStroke(attrs)}>
    {#each children ?? [] as child, i (i)}
      {@render node(child)}
    {/each}
  </svelte:element>
{/snippet}

<svg
  viewBox={icon.viewBox}
  width={size}
  height={size}
  fill="currentColor"
  role={title ? 'img' : undefined}
  aria-hidden={title ? undefined : true}
  aria-label={title}
>
  {#each icon.nodes as n, i (i)}
    {@render node(n)}
  {/each}
</svg>
