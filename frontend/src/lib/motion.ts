import { cubicIn, cubicOut } from 'svelte/easing'
import type { TransitionConfig } from 'svelte/transition'

import { settings } from './stores/settings.svelte'
import { genieCss } from './wm/genie'

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

/** How long the genie takes; DESIGN.md caps animations at 300ms. */
export const GENIE_MS = 300

/**
 * Window opening/closing: the genie, out of the app's dock tile and back
 * into it, like minimizing. Apps without a tile, and windows not on screen
 * (another workspace), get a short zoom and fade instead.
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
  const target = tile.getBoundingClientRect()
  // t runs 0→1 opening and 1→0 closing; the genie's own phases ease it.
  return { duration: ms(GENIE_MS), css: (t) => genieCss(box, target, 1 - t) }
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
