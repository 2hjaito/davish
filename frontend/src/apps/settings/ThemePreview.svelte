<script lang="ts">
  interface Props {
    theme: 'system' | 'light' | 'dark'
  }

  let { theme }: Props = $props()

  const uid = $props.id()
  const id = (name: string) => `${uid}-${name}`
</script>

<!--
  A small desktop in the chosen look: wallpaper, then a window with traffic
  lights, a sidebar and a few lines of text. Auto is light with its lower
  right half dark. Colors are artwork, not tokens: the preview must show
  the other theme whatever the current one is.
-->
{#snippet desk(dark: boolean)}
  <rect width="88" height="56" fill="url(#{id(dark ? 'wd' : 'wl')})" />
  <rect
    x="14"
    y="11"
    width="60"
    height="36"
    rx="4"
    fill={dark ? '#23252d' : '#ffffff'}
    stroke={dark ? '#3a3d48' : '#d9dce3'}
    stroke-width="0.6"
  />
  <path d="M18 11h12v36h-12a4 4 0 0 1-4-4V15a4 4 0 0 1 4-4z" fill={dark ? '#2e313b' : '#eef0f4'} />
  <circle cx="18.5" cy="15" r="1.3" fill="#ff5f57" />
  <circle cx="22.3" cy="15" r="1.3" fill="#febc2e" />
  <circle cx="26.1" cy="15" r="1.3" fill="#28c840" />
  <g fill={dark ? '#5b6070' : '#c9cdd6'}>
    <rect x="17" y="21" width="10" height="2" rx="1" />
    <rect x="17" y="26" width="8" height="2" rx="1" />
    <rect x="17" y="31" width="9" height="2" rx="1" />
    <rect x="35" y="21" width="30" height="2.4" rx="1.2" />
    <rect x="35" y="27" width="22" height="2.4" rx="1.2" />
    <rect x="35" y="33" width="26" height="2.4" rx="1.2" />
  </g>
{/snippet}

<svg viewBox="0 0 88 56" aria-hidden="true">
  <defs>
    <linearGradient id={id('wl')} x1="0" y1="0" x2="1" y2="1">
      <stop offset="0" stop-color="#7fb2ff" />
      <stop offset="1" stop-color="#2f6fe0" />
    </linearGradient>
    <linearGradient id={id('wd')} x1="0" y1="0" x2="1" y2="1">
      <stop offset="0" stop-color="#1f3d7a" />
      <stop offset="1" stop-color="#0b1730" />
    </linearGradient>
    <clipPath id={id('half')}>
      <path d="M88 0V56H30z" />
    </clipPath>
    <clipPath id={id('round')}>
      <rect width="88" height="56" rx="6" />
    </clipPath>
  </defs>
  <g clip-path="url(#{id('round')})">
    {@render desk(theme === 'dark')}
    {#if theme === 'system'}
      <g clip-path="url(#{id('half')})">{@render desk(true)}</g>
    {/if}
  </g>
</svg>

<style>
  svg {
    display: block;
    width: 88px;
    height: 56px;
  }
</style>
