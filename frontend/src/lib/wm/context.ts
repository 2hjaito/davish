import { getContext, setContext } from 'svelte'

import type { WindowState } from './windowManager.svelte'

export interface WindowContext {
  readonly win: WindowState
  readonly focused: boolean
  readonly theme: 'light' | 'dark'
  /** On screen right now: not minimized, on the active server, app not hidden. Pollers stop when false. */
  readonly showing: boolean
  startDrag(e: PointerEvent): void
  toggleMaximize(e: MouseEvent): void
}

const KEY = Symbol('window')

export function setWindowContext(ctx: WindowContext) {
  setContext(KEY, ctx)
}

export function getWindowContext(): WindowContext {
  const ctx = getContext<WindowContext | undefined>(KEY)
  if (!ctx) throw new Error('TitleBar must be used inside Window')
  return ctx
}
