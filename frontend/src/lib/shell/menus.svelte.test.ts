import { flushSync, untrack } from 'svelte'
import { describe, expect, it } from 'vitest'

import { appMenus, type MenuEntry, trackAppMenu } from './menus.svelte'

const labels = (entries: MenuEntry[]) => entries.map((x) => x?.label ?? '-')

/**
 * Runs setup as a component under {#if show()} would: set up untracked,
 * as Svelte sets up components, and torn down when show() turns false.
 */
function underIf(show: () => boolean, setup: () => () => void) {
  $effect.pre(() => {
    if (show()) return untrack(setup)
  })
}

describe('trackAppMenu', () => {
  // Docker's container page: the inspector's items read container.name, and
  // the page shows it under {#if container}. Going back clears container.
  it('lets a parent drop the props behind the items of a child it removes', () => {
    let container = $state<{ name: string } | undefined>({ name: 'web' })
    let shown: string[] = []
    const stop = $effect.root(() => {
      // The menu bar, set up before the windows as on the desktop.
      $effect.pre(() => {
        shown = labels(appMenus.items('w-docker'))
      })
      underIf(
        () => container !== undefined,
        () =>
          trackAppMenu('w-docker', () => [{ label: `Stop ${container!.name}…`, run: () => {} }]),
      )
    })
    try {
      flushSync()
      expect(shown).toEqual(['Stop web…'])

      container = undefined
      expect(() => flushSync()).not.toThrow()
      expect(shown).toEqual([])
    } finally {
      stop()
    }
  })

  it('follows what the items read, and forgets them with the child', () => {
    let busy = $state(false)
    let open = $state(true)
    let shown: string[] = []
    const stop = $effect.root(() => {
      $effect.pre(() => {
        shown = labels(appMenus.items('w-busy'))
      })
      underIf(
        () => open,
        () => trackAppMenu('w-busy', () => [{ label: busy ? 'busy' : 'idle', run: () => {} }]),
      )
    })
    try {
      flushSync()
      expect(shown).toEqual(['idle'])
      busy = true
      flushSync()
      expect(shown).toEqual(['busy'])
      open = false
      flushSync()
      expect(shown).toEqual([])
    } finally {
      stop()
    }
  })
})
