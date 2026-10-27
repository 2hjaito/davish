<script lang="ts">
  interface Props {
    size?: number
    /** Rounded tile behind the mark; off when the shape comes from elsewhere. */
    tile?: boolean
  }

  let { size = 64, tile = true }: Props = $props()

  const uid = $props.id()
  const ref = (name: string) => `${uid}-${name}`
  const url = (name: string) => `url(#${ref(name)})`
</script>

<!--
  The Davish mark (docs/DESIGN.md, "Icon Davish"): a desktop window holding
  a server, on a blue-to-violet tile. Colors are artwork, not UI tokens.
-->
<svg viewBox="0 0 64 64" width={size} height={size} aria-hidden="true">
  <defs>
    <linearGradient id={ref('bg')} x1="0" y1="0" x2="1" y2="1">
      <stop offset="0" stop-color="#5b8cff" />
      <stop offset="1" stop-color="#6a2fe0" />
    </linearGradient>
    <linearGradient id={ref('win')} x1="0" y1="0" x2="0" y2="1">
      <stop offset="0" stop-color="#ffffff" />
      <stop offset="1" stop-color="#e9edff" />
    </linearGradient>
  </defs>
  {#if tile}<rect width="64" height="64" rx="14.5" fill={url('bg')} />{/if}
  <rect x="9" y="12.5" width="46" height="41" rx="8" fill="#22106b" opacity="0.28" />
  <rect x="9" y="10.5" width="46" height="41" rx="8" fill={url('win')} />
  <circle cx="15.5" cy="16.5" r="1.9" fill="#ff5f57" />
  <circle cx="21" cy="16.5" r="1.9" fill="#febc2e" />
  <circle cx="26.5" cy="16.5" r="1.9" fill="#28c840" />
  {#each [24, 32.5, 41] as y (y)}
    <rect x="17" {y} width="30" height="6.5" rx="2.4" fill="#4b3bd1" />
    <circle cx="42" cy={y + 3.25} r="1.5" fill="#5ef0a0" />
    <rect x="20.5" y={y + 2.4} width="9" height="1.7" rx="0.85" fill="#a99cff" />
  {/each}
</svg>
