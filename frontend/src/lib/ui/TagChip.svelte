<script lang="ts">
  interface Props {
    tag: string
    /** The server's color; without one the chip stays neutral. */
    color?: string
    /** Filled, for the menu bar where it must not be missed. */
    solid?: boolean
  }

  let { tag, color, solid = false }: Props = $props()
</script>

<span class="tag" class:solid class:plain={!color} style:--c={color} title={tag}>{tag}</span>

<style>
  .tag {
    display: inline-block;
    flex-shrink: 0;
    max-width: 120px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    padding: 2px 7px;
    border-radius: 999px;
    font-size: 10px;
    font-weight: 700;
    line-height: 1.2;
    letter-spacing: 0.05em;
    text-transform: uppercase;
    background: color-mix(in srgb, var(--c) 16%, transparent);
    /* Mixed with the text color so a light orange still reads on glass. */
    color: color-mix(in srgb, var(--c) 72%, var(--fg));
  }

  .tag.plain {
    background: var(--fill-soft);
    color: var(--fg-muted);
  }

  .tag.solid:not(.plain) {
    background: var(--c);
    color: #fff;
  }
</style>
