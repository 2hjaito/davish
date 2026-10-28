import { cubicIn, cubicOut } from 'svelte/easing'
import type { TransitionConfig } from 'svelte/transition'

import { settings } from './stores/settings.svelte'
import { centerOffset } from './wm/geometry'

const prefersReduced =
  typeof matchMedia !== 'undefined' && matchMedia('(prefers-reduced-motion: reduce)').matches

/** Duration for JS-driven transitions; 0 when the user asked for less motion. */
export function ms(duration: number): number {
  return settings.reduceMotion || prefersReduced ? 0 : duration
}

/** The app's tile in the dock, which its windows fly from and back to. */
export function dockTile(appId: string): Element | null {
  return document.querySelector(`[data-dock-app="${appId}"]`)
}

/**
 * Window opening/closing: grows out of the app's dock tile and shrinks
 * back into it, like minimizing. Apps without a tile, and windows not on
 * screen (another workspace), get a short zoom and fade instead.
 */
export function windowZoom(node: Element, { out = false, appId = '' } = {}): TransitionConfig {
  const tile = appId ? dockTile(appId) : null
  const box = node.getBoundingClientRect()
  if (!tile || box.width === 0) {
    return {
      duration: ms(out ? 140 : 200),
      easing: out ? cubicIn : cubicOut,
      css: (t) => `opacity: ${t}; transform: scale(${0.94 + 0.06 * t})`,
    }
  }
  const { dx, dy } = centerOffset(box, tile.getBoundingClientRect())
  return {
    duration: ms(out ? 200 : 260),
    easing: out ? cubicIn : cubicOut,
    css: (t, u) =>
      `opacity: ${t}; transform: translate(${dx * u}px, ${dy * u}px) scale(${0.12 + 0.88 * t})`,
  }
}

/** Menus and popovers drop in from just above their anchor. */
export function popIn(_node: Element): TransitionConfig {
  return {
    duration: ms(140),
    easing: cubicOut,
    css: (t) => `opacity: ${t}; transform: translateY(${(1 - t) * -4}px) scale(${0.98 + 0.02 * t})`,
  }
}

export function fadeOut(_node: Element): TransitionConfig {
  return { duration: ms(100), css: (t) => `opacity: ${t}` }
}
