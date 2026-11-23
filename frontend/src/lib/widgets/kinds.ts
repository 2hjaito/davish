// The kinds of widget, apart from their components so the store and the
// layout can know them without loading any. A kind is added here and in
// registry.ts together (ADR 0026).
export const WIDGET_KINDS = ['monitor', 'docker'] as const

export type WidgetKind = (typeof WIDGET_KINDS)[number]

export function isWidgetKind(v: unknown): v is WidgetKind {
  return WIDGET_KINDS.some((k) => k === v)
}
