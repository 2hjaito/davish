<script lang="ts">
  import { areaPath, linePath, type Point } from '../ui/charts/scale'

  interface Series {
    values: number[]
    /** A CSS color. */
    color: string
    /** Fill under the line. */
    area?: boolean
  }

  interface Props {
    series: Series[]
    times: number[]
    /** The value at the top of the chart. */
    max: number
  }

  let { series, times, max }: Props = $props()

  const uid = $props.id()

  const W = 100
  const H = 28
  /** Two samples further apart than this leave a hole, as in the charts. */
  const MAX_GAP_MS = 10_000

  const drawn = $derived.by(() => {
    if (times.length < 2 || max <= 0) return []
    const first = times[0] ?? 0
    const span = Math.max(1, (times[times.length - 1] ?? 0) - first)
    return series.map((s) => {
      const points: Point[] = s.values.map((v, i) => ({
        x: (((times[i] ?? 0) - first) / span) * W,
        y: H - 1.5 - Math.min(1, Math.max(0, v / max)) * (H - 3),
      }))
      return {
        color: s.color,
        line: linePath(points, times, MAX_GAP_MS),
        fill: s.area ? areaPath(points, times, MAX_GAP_MS, H) : '',
      }
    })
  })
</script>

<svg viewBox="0 0 {W} {H}" preserveAspectRatio="none" aria-hidden="true">
  <defs>
    {#each drawn as d, i (i)}
      <linearGradient id="{uid}-{i}" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" stop-color={d.color} stop-opacity="0.45" />
        <stop offset="1" stop-color={d.color} stop-opacity="0" />
      </linearGradient>
    {/each}
  </defs>
  {#each drawn as d, i (i)}
    {#if d.fill}<path d={d.fill} fill="url(#{uid}-{i})" stroke="none" />{/if}
    <path
      d={d.line}
      fill="none"
      stroke={d.color}
      stroke-width="1.6"
      stroke-linejoin="round"
      vector-effect="non-scaling-stroke"
    />
  {/each}
</svg>

<style>
  svg {
    display: block;
    width: 100%;
    height: 100%;
  }
</style>
