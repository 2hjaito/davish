<script lang="ts">
  import { linePath, type Point } from '../ui/charts/scale'

  interface Props {
    values: number[]
    times: number[]
    /** The value at the top of the chart. */
    max: number
    /** A CSS color. */
    color?: string
  }

  let { values, times, max, color = 'var(--chart-1)' }: Props = $props()

  const W = 100
  const H = 24
  /** Two samples further apart than this leave a hole, as in the charts. */
  const MAX_GAP_MS = 10_000

  const d = $derived.by(() => {
    if (values.length < 2 || max <= 0) return ''
    const first = times[0] ?? 0
    const span = Math.max(1, (times[times.length - 1] ?? 0) - first)
    const points: Point[] = values.map((v, i) => ({
      x: (((times[i] ?? 0) - first) / span) * W,
      y: H - 1 - Math.min(1, Math.max(0, v / max)) * (H - 2),
    }))
    return linePath(points, times, MAX_GAP_MS)
  })
</script>

<svg viewBox="0 0 {W} {H}" preserveAspectRatio="none" aria-hidden="true">
  <path {d} fill="none" stroke={color} stroke-width="1.5" vector-effect="non-scaling-stroke" />
</svg>

<style>
  svg {
    display: block;
    width: 100%;
    height: 100%;
  }
</style>
