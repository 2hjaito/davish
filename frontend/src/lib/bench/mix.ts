// Which widgets the render bench puts on the desktop to measure what they
// cost: the medium Monitor (three charts), the medium Docker (a list) and the
// small Monitor, then the same again. Heavy kinds come first, so one widget is
// the most expensive one.
import type { WidgetKind } from '../widgets/kinds'
import type { WidgetSize } from '../widgets/layout'

const MIX: readonly (readonly [WidgetKind, WidgetSize])[] = [
  ['monitor', 'M'],
  ['docker', 'M'],
  ['monitor', 'S'],
]

/** The first n widgets of the mix, repeating it when n is more than it holds. */
export function widgetMix(n: number): [WidgetKind, WidgetSize][] {
  return Array.from({ length: Math.max(0, Math.floor(n)) }, (_, i) => {
    const [kind, size] = MIX[i % MIX.length]!
    return [kind, size]
  })
}
