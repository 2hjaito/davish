<script lang="ts">
  import AppIcon from '../../lib/ui/AppIcon.svelte'
  import type { Pane } from './panes'

  interface Props {
    pane: Pane
    size?: number
  }

  let { pane, size = 22 }: Props = $props()

  // Panes that are about an app wear that app's icon.
  const APP: Partial<Record<Pane, 'settings' | 'terminal' | 'editor'>> = {
    general: 'settings',
    terminal: 'terminal',
    editor: 'editor',
  }

  // The rest are drawn by the app icon rules (ADR 0003): one bold filled
  // shape on a rich vertical gradient. Colors are artwork, not tokens.
  const BG: Record<Pane, [string, string]> = {
    general: ['#f5f6f9', '#b3b7c2'],
    appearance: ['#8a93ff', '#3a3fd0'],
    performance: ['#ffbe4d', '#f06a00'],
    menubar: ['#6ad3ff', '#1a8ae0'],
    dock: ['#5ab1ff', '#1d58d4'],
    terminal: ['#3b404c', '#121418'],
    editor: ['#ff5c8f', '#d9103c'],
    alerts: ['#ff7a70', '#d9231a'],
    security: ['#b88cff', '#6a2ee0'],
  }

  const uid = $props.id()
  const ref = (name: string) => `${uid}-${name}`
  const url = (name: string) => `url(#${ref(name)})`
  const app = $derived(APP[pane])
  const [top, bottom] = $derived(BG[pane])
</script>

<span class="tile" style:--size="{size}px">
  {#if app}
    <AppIcon id={app} {size} />
  {:else}
    <svg viewBox="0 0 64 64" width={size} height={size} aria-hidden="true">
      <defs>
        <linearGradient id={ref('bg')} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stop-color={top} />
          <stop offset="1" stop-color={bottom} />
        </linearGradient>
      </defs>
      <rect width="64" height="64" fill={url('bg')} />
      {#if pane === 'appearance'}
        <!-- Light and dark halves of one disc. -->
        <circle cx="32" cy="32" r="20" fill="#fff" />
        <path d="M32 12a20 20 0 0 1 0 40Z" fill="#1d1f27" />
      {:else if pane === 'performance'}
        <!-- A gauge, needle well up. -->
        <path
          d="M14.5 42a19 19 0 1 1 35 0"
          fill="none"
          stroke="#fff"
          stroke-width="7"
          stroke-linecap="round"
        />
        <path
          d="M32 39 43 26"
          fill="none"
          stroke="#7a3300"
          stroke-width="5"
          stroke-linecap="round"
        />
        <circle cx="32" cy="39" r="5" fill="#7a3300" />
      {:else if pane === 'menubar'}
        <!-- A screen with its bar along the top. -->
        <rect x="10" y="14" width="44" height="36" rx="6" fill="#fff" />
        <path d="M16 14h32a6 6 0 0 1 6 6v3H10v-3a6 6 0 0 1 6-6Z" fill="#0b5fa8" />
        <rect x="15" y="17" width="8" height="3" rx="1.5" fill="#fff" />
        <rect x="41" y="17" width="8" height="3" rx="1.5" fill="#fff" opacity="0.7" />
      {:else if pane === 'dock'}
        <!-- A screen with its Dock of tiles. -->
        <rect x="10" y="14" width="44" height="36" rx="6" fill="#bfdcff" />
        <rect x="17" y="38" width="30" height="9" rx="3.5" fill="#fff" />
        <rect x="20" y="40" width="5" height="5" rx="1.5" fill="#ff8a00" />
        <rect x="29.5" y="40" width="5" height="5" rx="1.5" fill="#1d58d4" />
        <rect x="39" y="40" width="5" height="5" rx="1.5" fill="#28b04e" />
      {:else if pane === 'alerts'}
        <!-- A bell. -->
        <path
          d="M32 12a3 3 0 0 1 3 3v1.3A13 13 0 0 1 45 29v9l4 5v2H15v-2l4-5v-9a13 13 0 0 1 10-12.7V15a3 3 0 0 1 3-3Z"
          fill="#fff"
        />
        <path d="M26.5 48h11a5.5 5.5 0 0 1-11 0Z" fill="#ffd6d2" />
      {:else if pane === 'security'}
        <!-- A padlock. -->
        <path
          d="M22 30v-6a10 10 0 0 1 20 0v6"
          fill="none"
          stroke="#e6d8ff"
          stroke-width="6"
          stroke-linecap="round"
        />
        <rect x="15" y="28" width="34" height="24" rx="6" fill="#fff" />
        <circle cx="32" cy="38" r="3.5" fill="#6a2ee0" />
        <rect x="30.5" y="39" width="3" height="7" rx="1.5" fill="#6a2ee0" />
      {/if}
    </svg>
  {/if}
</span>

<style>
  /* The Dock's finish, scaled down: rounded corners, top sheen, thin edge. */
  .tile {
    position: relative;
    flex-shrink: 0;
    display: block;
    width: var(--size);
    height: var(--size);
    border-radius: calc(var(--size) * 0.27);
    overflow: hidden;
    box-shadow: 0 0.5px 1.5px rgba(0, 0, 0, 0.3);
  }

  .tile :global(svg) {
    display: block;
  }

  .tile::after {
    content: '';
    position: absolute;
    inset: 0;
    border-radius: inherit;
    background: var(--tile-sheen);
    box-shadow: var(--tile-edge);
    pointer-events: none;
  }
</style>
