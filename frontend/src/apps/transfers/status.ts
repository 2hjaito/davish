// The status line under each transfer.
import { formatBytes, formatDuration, formatRate } from '../../lib/format'
import { t } from '../../lib/i18n'
import type { Transfer } from '../../lib/stores/transfers.svelte'

/** Time left at the current speed, or '' when it cannot be told. */
export function timeLeft(j: Pick<Transfer, 'done' | 'total' | 'bps'>): string {
  if (j.bps <= 0 || j.total <= 0 || j.done >= j.total) return ''
  const s = Math.ceil((j.total - j.done) / j.bps)
  return s < 60 ? t.transfers.seconds(s) : formatDuration(s, t.units)
}

export function statusText(j: Transfer): string {
  switch (j.state) {
    case 'done':
      return j.total > 0 ? `${t.transfers.done} · ${formatBytes(j.total)}` : t.transfers.done
    case 'canceled':
      return t.transfers.canceled
    case 'failed':
      return j.error ?? ''
  }
  // Waiting for a free slot, or walking a folder to learn its size.
  if (j.done === 0 && j.total === 0) return t.transfers.waiting
  const parts = [
    j.total > 0 ? `${formatBytes(j.done)} / ${formatBytes(j.total)}` : formatBytes(j.done),
  ]
  if (j.bps > 0) parts.push(formatRate(j.bps))
  const left = timeLeft(j)
  if (left) parts.push(t.transfers.left(left))
  return parts.join(' · ')
}

/** 0 to 100, for the progress bar. */
export function percent(j: Transfer): number {
  if (j.state === 'done') return 100
  return j.total > 0 ? Math.min(100, (j.done / j.total) * 100) : 0
}
