<script lang="ts">
  interface Props {
    size?: number
  }

  let { size = 64 }: Props = $props()

  // Gradient and filter ids must be unique per instance.
  const uid = $props.id()
  const ref = (name: string) => `${uid}-${name}`
  const url = (name: string) => `url(#${ref(name)})`
</script>

<!--
  The Davish icon (assets/icon/davish-icon.svg, the source of build/appicon.png):
  three server bars on a blue tile, the middle one solid with a green light.
  Colors are artwork, not UI tokens. Keep in step with the asset.
-->
<svg viewBox="0 0 100 100" width={size} height={size} aria-hidden="true">
  <defs>
    <linearGradient id={ref('bg')} x1="0" y1="0" x2="0" y2="1">
      <stop offset="0" stop-color="#4a96ff" />
      <stop offset="0.55" stop-color="#1a6ff0" />
      <stop offset="1" stop-color="#0a4cc2" />
    </linearGradient>
    <radialGradient id={ref('sheen')} cx="0.3" cy="0.12" r="0.75">
      <stop offset="0" stop-color="#ffffff" stop-opacity="0.38" />
      <stop offset="0.6" stop-color="#ffffff" stop-opacity="0.05" />
      <stop offset="1" stop-color="#ffffff" stop-opacity="0" />
    </radialGradient>
    <linearGradient id={ref('rim')} x1="0" y1="0" x2="1" y2="1">
      <stop offset="0" stop-color="#ffffff" stop-opacity="0.85" />
      <stop offset="0.45" stop-color="#ffffff" stop-opacity="0.18" />
      <stop offset="1" stop-color="#ffffff" stop-opacity="0.45" />
    </linearGradient>
    <linearGradient id={ref('glassFill')} x1="0" y1="0" x2="0" y2="1">
      <stop offset="0" stop-color="#ffffff" stop-opacity="0.42" />
      <stop offset="1" stop-color="#ffffff" stop-opacity="0.14" />
    </linearGradient>
    <linearGradient id={ref('glassRim')} x1="0" y1="0" x2="1" y2="1">
      <stop offset="0" stop-color="#ffffff" stop-opacity="0.95" />
      <stop offset="0.5" stop-color="#ffffff" stop-opacity="0.35" />
      <stop offset="1" stop-color="#ffffff" stop-opacity="0.7" />
    </linearGradient>
    <linearGradient id={ref('solid')} x1="0" y1="0" x2="0" y2="1">
      <stop offset="0" stop-color="#ffffff" />
      <stop offset="1" stop-color="#e3edff" />
    </linearGradient>
    <filter id={ref('barShadow')} x="-30%" y="-60%" width="160%" height="260%">
      <feDropShadow dx="0" dy="1.6" stdDeviation="1.6" flood-color="#06307e" flood-opacity="0.38" />
    </filter>
    <filter id={ref('glow')} x="-200%" y="-200%" width="500%" height="500%">
      <feGaussianBlur stdDeviation="1.1" result="b" />
      <feMerge><feMergeNode in="b" /><feMergeNode in="SourceGraphic" /></feMerge>
    </filter>
    <clipPath id={ref('sq')}><rect width="100" height="100" rx="22.5" /></clipPath>
  </defs>
  <g clip-path={url('sq')}>
    <rect width="100" height="100" fill={url('bg')} />
    <rect width="100" height="100" fill={url('sheen')} />
  </g>
  <rect
    x="0.6"
    y="0.6"
    width="98.8"
    height="98.8"
    rx="21.9"
    fill="none"
    stroke={url('rim')}
    stroke-width="1.2"
  />

  <g filter={url('barShadow')}>
    <rect x="22" y="24" width="40" height="13" rx="6.5" fill={url('glassFill')} />
    <rect x="22" y="63" width="40" height="13" rx="6.5" fill={url('glassFill')} />
    <rect x="22" y="43.5" width="56" height="13" rx="6.5" fill={url('solid')} />
  </g>
  <rect
    x="22.4"
    y="24.4"
    width="39.2"
    height="12.2"
    rx="6.1"
    fill="none"
    stroke={url('glassRim')}
    stroke-width="0.8"
  />
  <rect
    x="22.4"
    y="63.4"
    width="39.2"
    height="12.2"
    rx="6.1"
    fill="none"
    stroke={url('glassRim')}
    stroke-width="0.8"
  />
  <rect x="34" y="25.3" width="24" height="2.2" rx="1.1" fill="#ffffff" fill-opacity="0.45" />
  <rect x="34" y="64.3" width="24" height="2.2" rx="1.1" fill="#ffffff" fill-opacity="0.45" />

  <g filter={url('glow')}>
    <circle cx="29" cy="30.5" r="2.5" fill="#ffffff" />
    <circle cx="29" cy="69.5" r="2.5" fill="#ffffff" />
    <circle cx="29" cy="50" r="2.6" fill="#22c55e" />
  </g>
</svg>
