<script lang="ts">
  // The logo of the system a server runs (Ubuntu, Debian, Windows, macOS…),
  // beside its name. A server whose system is not known yet gets a plain
  // server glyph, and the box keeps its size while the logos load, so the
  // name does not jump when the logo arrives.
  import type { IconDef } from './icons'
  import Icon from './Icon.svelte'
  import { osLogos } from './osIcons.svelte'

  interface Props {
    /** The server's logo key (store.Server.Distro). */
    distro?: string
    size?: number
    /** Draw it in the text color, for a background the brand colors do not suit. */
    mono?: boolean
    /** A box `size` wide with a larger glyph, for beside two lines of text. */
    tile?: boolean
  }

  let { distro, size = 16, mono = false, tile = false }: Props = $props()

  $effect(() => {
    if (distro) osLogos.load()
  })

  const glyph = $derived(tile ? Math.round(size * 0.8) : size)
  const logo = $derived(
    distro && osLogos.defs
      ? ((osLogos.defs as Record<string, IconDef & { color: string | null }>)[distro] ?? null)
      : null,
  )
</script>

<span
  class="os"
  class:unknown={!distro}
  class:tile
  style:width="{size}px"
  style:height="{size}px"
  style:color={mono ? 'inherit' : (logo?.color ?? null)}
>
  {#if logo}
    <Icon def={logo} size={glyph} />
  {:else if !distro}
    <Icon name="server" size={tile ? glyph : size - 2} />
  {/if}
</span>

<style>
  .os {
    flex-shrink: 0;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    color: var(--fg-muted);
  }

  .unknown {
    opacity: 0.55;
  }
</style>
