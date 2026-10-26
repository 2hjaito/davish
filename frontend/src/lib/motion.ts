import { cubicIn, cubicOut } from 'svelte/easing'
import type { TransitionConfig } from 'svelte/transition'

import { settings } from './stores/settings.svelte'

const prefersReduced =
  typeof matchMedia !== 'undefined' && matchMedia('(prefers-reduced-motion: reduce)').matches

/** Duration for JS-driven transitions; 0 when the user asked for less motion. */
export function ms(duration: number): number {
  return settings.reduceMotion || prefersReduced ? 0 : duration
}

/** Window opening/closing: a short zoom and fade, like macOS. */
export function windowZoom(_node: Element, { out = false } = {}): TransitionConfig {
  return {
    duration: ms(out ? 140 : 200),
    easing: out ? cubicIn : cubicOut,
    css: (t) => `opacity: ${t}; transform: scale(${0.94 + 0.06 * t})`,
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
