<script lang="ts">
  import type { AppId } from '../../apps/types'
  import Icon from './Icon.svelte'

  interface Props {
    id: AppId | 'launchpad'
    size?: number
  }

  let { id, size = 56 }: Props = $props()

  // Gradient ids must be unique per instance, or icons rendered twice would
  // reference each other's definitions.
  const uid = $props.id()
  const ref = (name: string) => `${uid}-${name}`
  const url = (name: string) => `url(#${ref(name)})`
</script>

<!--
  App icons (docs/adr/0003-app-icon-artwork.md) on a 64×64 grid. Davish's own
  apps get drawn artwork: one bold, filled shape on a rich gradient. Apps that
  manage a third-party tool show that tool's official mark. Colors here are
  artwork, not UI tokens.
-->
<svg viewBox="0 0 64 64" width={size} height={size} aria-hidden="true">
  {#if id === 'terminal'}
    <defs>
      <linearGradient id={ref('bg')} x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" stop-color="#3b404c" />
        <stop offset="1" stop-color="#121418" />
      </linearGradient>
    </defs>
    <rect width="64" height="64" fill={url('bg')} />
    <rect x="9" y="14" width="46" height="36" rx="10" fill="#08090c" />
    <rect x="9.5" y="14.5" width="45" height="35" rx="9.5" fill="none" stroke="#ffffff14" />
    <path
      d="M19 24.5 27.5 32 19 39.5"
      fill="none"
      stroke="#5ef0a0"
      stroke-width="5.5"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
    <rect x="31" y="36.5" width="14" height="5.5" rx="2.75" fill="#fff" />
  {:else if id === 'files'}
    <defs>
      <linearGradient id={ref('bg')} x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" stop-color="#5ab1ff" />
        <stop offset="1" stop-color="#1d58d4" />
      </linearGradient>
    </defs>
    <rect width="64" height="64" fill={url('bg')} />
    <path
      d="M11 21a5 5 0 0 1 5-5h10.5a4 4 0 0 1 3 1.3l3.2 3.7H48a5 5 0 0 1 5 5v20a5 5 0 0 1-5 5H16a5 5 0 0 1-5-5Z"
      fill="#bfdcff"
    />
    <rect x="11" y="28" width="42" height="24" rx="5" fill="#0b3a9e" opacity="0.25" />
    <rect x="11" y="26" width="42" height="25" rx="5" fill="#fff" />
  {:else if id === 'monitor'}
    <defs>
      <linearGradient id={ref('bg')} x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" stop-color="#22242b" />
        <stop offset="1" stop-color="#050506" />
      </linearGradient>
      <radialGradient id={ref('glow')} cx="0.2" cy="1" r="0.9">
        <stop offset="0" stop-color="#ff5a1f" stop-opacity="0.55" />
        <stop offset="1" stop-color="#ff5a1f" stop-opacity="0" />
      </radialGradient>
      <linearGradient id={ref('line')} x1="0" y1="0" x2="1" y2="0">
        <stop offset="0" stop-color="#ffd23f" />
        <stop offset="0.55" stop-color="#ff5a1f" />
        <stop offset="1" stop-color="#ff2d6f" />
      </linearGradient>
      <filter id={ref('blur')} x="-20%" y="-20%" width="140%" height="140%">
        <feGaussianBlur stdDeviation="2.5" />
      </filter>
    </defs>
    <rect width="64" height="64" fill={url('bg')} />
    <rect width="64" height="64" fill={url('glow')} />
    {#each [true, false] as glow (glow)}
      <path
        d="M9 35h9.5l5.5-14 8.5 25 6-16 3.5 5H55"
        fill="none"
        stroke={url('line')}
        stroke-width="6"
        stroke-linecap="round"
        stroke-linejoin="round"
        filter={glow ? url('blur') : undefined}
        opacity={glow ? 0.7 : 1}
      />
    {/each}
  {:else if id === 'services'}
    <defs>
      <linearGradient id={ref('bg')} x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" stop-color="#ffc04a" />
        <stop offset="1" stop-color="#e2670c" />
      </linearGradient>
    </defs>
    <rect width="64" height="64" fill={url('bg')} />
    {#each [{ y: 39, o: 0.5 }, { y: 31, o: 0.75 }, { y: 23, o: 1 }] as layer (layer.y)}
      <path
        d="M32 {layer.y - 10} 52 {layer.y} 32 {layer.y + 10} 12 {layer.y}Z"
        fill="#fff"
        fill-opacity={layer.o}
        stroke="#ef8a1c"
        stroke-width="2.5"
        stroke-linejoin="round"
      />
    {/each}
  {:else if id === 'docker'}
    <!-- Third-party tool: its official mark, unmodified, in its brand color (ADR 0003). -->
    <defs>
      <linearGradient id={ref('bg')} x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" stop-color="#ffffff" />
        <stop offset="1" stop-color="#e3ecf7" />
      </linearGradient>
    </defs>
    <rect width="64" height="64" fill={url('bg')} />
    <g transform="translate(12 12)" style:color="#2496ed">
      <Icon name="brandDocker" size={40} />
    </g>
  {:else if id === 'nginx'}
    <!-- Third-party tool: its official mark, unmodified, in its brand color (ADR 0003). -->
    <defs>
      <linearGradient id={ref('bg')} x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" stop-color="#ffffff" />
        <stop offset="1" stop-color="#e2f1e6" />
      </linearGradient>
    </defs>
    <rect width="64" height="64" fill={url('bg')} />
    <g transform="translate(13 13)" style:color="#009639">
      <Icon name="brandNginx" size={38} />
    </g>
  {:else if id === 'cron'}
    <defs>
      <linearGradient id={ref('bg')} x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" stop-color="#4fe0d2" />
        <stop offset="1" stop-color="#0a8791" />
      </linearGradient>
    </defs>
    <rect width="64" height="64" fill={url('bg')} />
    <circle cx="32" cy="34" r="21" fill="#06555c" opacity="0.3" />
    <circle cx="32" cy="32" r="21" fill="#fff" />
    {#each [0, 90, 180, 270] as angle (angle)}
      <rect
        x="30.5"
        y="14"
        width="3"
        height="5"
        rx="1.5"
        fill="#9ad9d6"
        transform="rotate({angle} 32 32)"
      />
    {/each}
    <path
      d="M32 20.5V32l8 5"
      fill="none"
      stroke="#0b5f66"
      stroke-width="4.5"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
    <circle cx="32" cy="32" r="3.2" fill="#ffc83d" />
  {:else if id === 'logs'}
    <defs>
      <linearGradient id={ref('bg')} x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" stop-color="#b18bff" />
        <stop offset="1" stop-color="#5426cc" />
      </linearGradient>
    </defs>
    <rect width="64" height="64" fill={url('bg')} />
    <g transform="rotate(-7 32 32)">
      <rect x="15" y="11" width="34" height="45" rx="7" fill="#2b0f7a" opacity="0.3" />
      <rect x="15" y="9" width="34" height="45" rx="7" fill="#fff" />
      <rect x="21" y="18" width="22" height="4.5" rx="2.25" fill="#d3c4fb" />
      <rect x="21" y="27" width="17" height="4.5" rx="2.25" fill="#ffc83d" />
      <rect x="21" y="36" width="20" height="4.5" rx="2.25" fill="#d3c4fb" />
      <rect x="21" y="45" width="12" height="4.5" rx="2.25" fill="#d3c4fb" />
    </g>
  {:else if id === 'editor'}
    <defs>
      <linearGradient id={ref('bg')} x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" stop-color="#ff5c8f" />
        <stop offset="1" stop-color="#d9103c" />
      </linearGradient>
    </defs>
    <rect width="64" height="64" fill={url('bg')} />
    <g transform="rotate(40 32 32)">
      <rect x="25.5" y="6" width="13" height="38" rx="3.5" fill="#fff" />
      <rect x="25.5" y="6" width="13" height="8" rx="3.5" fill="#ffc2d3" />
      <path d="M25.5 44h13L32 57Z" fill="#ffe0c2" stroke="#ffe0c2" stroke-linejoin="round" />
      <path d="M29.6 52.2h4.8L32 57Z" fill="#2b2b33" />
    </g>
  {:else if id === 'settings'}
    <defs>
      <linearGradient id={ref('bg')} x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" stop-color="#f5f6f9" />
        <stop offset="1" stop-color="#b3b7c2" />
      </linearGradient>
    </defs>
    <rect width="64" height="64" fill={url('bg')} />
    <g fill="#474b56">
      {#each [0, 45, 90, 135, 180, 225, 270, 315] as angle (angle)}
        <rect x="27.5" y="10" width="9" height="12" rx="2.5" transform="rotate({angle} 32 32)" />
      {/each}
      <circle cx="32" cy="32" r="16" />
    </g>
    <circle cx="32" cy="32" r="6.5" fill="#dfe1e7" />
  {:else if id === 'transfers'}
    <defs>
      <linearGradient id={ref('bg')} x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" stop-color="#5ec8ff" />
        <stop offset="1" stop-color="#1268e0" />
      </linearGradient>
    </defs>
    <rect width="64" height="64" fill={url('bg')} />
    <g fill="none" stroke="#fff" stroke-width="6" stroke-linecap="round" stroke-linejoin="round">
      <path d="M23 13v34M13 37l10 10 10-10" />
      <path d="M41 51V17M31 27l10-10 10 10" opacity="0.75" />
    </g>
  {:else if id === 'audit'}
    <defs>
      <linearGradient id={ref('bg')} x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" stop-color="#ffcf5c" />
        <stop offset="1" stop-color="#e8860c" />
      </linearGradient>
    </defs>
    <rect width="64" height="64" fill={url('bg')} />
    <rect x="14" y="12" width="36" height="44" rx="6" fill="#8a4a00" opacity="0.25" />
    <rect x="14" y="10" width="36" height="44" rx="6" fill="#fff" />
    <rect x="24" y="6" width="16" height="9" rx="3" fill="#5a3300" />
    {#each [22, 31, 40] as y (y)}
      <path
        d="M20 {y + 2}l2.5 2.5 4.5-5"
        fill="none"
        stroke="#e8860c"
        stroke-width="2.6"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
      <rect x="30" {y} width="14" height="4.5" rx="2.25" fill="#f2d9b3" />
    {/each}
  {:else if id === 'keys'}
    <defs>
      <linearGradient id={ref('bg')} x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" stop-color="#2f6f73" />
        <stop offset="1" stop-color="#123c40" />
      </linearGradient>
      <linearGradient id={ref('gold')} x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" stop-color="#ffe08a" />
        <stop offset="1" stop-color="#e8a317" />
      </linearGradient>
    </defs>
    <rect width="64" height="64" fill={url('bg')} />
    <g transform="rotate(-45 32 32)" fill={url('gold')}>
      <circle cx="20" cy="32" r="11" />
      <rect x="27" y="28.5" width="27" height="7" rx="2" />
      <rect x="44" y="35" width="5" height="7" rx="1.5" />
      <rect x="37" y="35" width="4.5" height="5" rx="1.5" />
    </g>
    <circle cx="23.5" cy="40.5" r="4.2" fill="#123c40" />
  {:else if id === 'packages'}
    <defs>
      <linearGradient id={ref('bg')} x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" stop-color="#ffb35c" />
        <stop offset="1" stop-color="#d9671a" />
      </linearGradient>
    </defs>
    <rect width="64" height="64" fill={url('bg')} />
    <path d="M32 12 51 22v21L32 53 13 43V22Z" fill="#fff4e3" />
    <path d="M32 33 51 22v21L32 53Z" fill="#f2c79a" />
    <path d="M32 33 13 22v21l19 10Z" fill="#fde3c4" />
    <path d="M22.5 17 41.5 27.5v7l-4-2v-5.3L18.5 19Z" fill="#b8540f" opacity="0.85" />
  {:else if id === 'launchpad'}
    <defs>
      <linearGradient id={ref('bg')} x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" stop-color="#8a93a8" />
        <stop offset="1" stop-color="#3d4456" />
      </linearGradient>
    </defs>
    <rect width="64" height="64" fill={url('bg')} />
    {#each [['#ff6b6b', '#ffc83d', '#4cd97b'], ['#3dc2ff', '#b18bff', '#ff5c8f'], ['#ffffff', '#5ec8ff', '#ffa23d']] as row, r (r)}
      {#each row as color, c (c)}
        <rect x={12 + c * 15} y={12 + r * 15} width="11" height="11" rx="3.5" fill={color} />
      {/each}
    {/each}
  {/if}
</svg>
