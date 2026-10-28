<script lang="ts">
  import { untrack } from 'svelte'

  import Desktop from './lib/shell/Desktop.svelte'
  import LockScreen from './lib/shell/LockScreen.svelte'
  import { appLock } from './lib/stores/lock.svelte'
  import { settings } from './lib/stores/settings.svelte'

  $effect(() => {
    const root = document.documentElement
    root.dataset.reduceGlass = String(settings.reduceGlass)
    root.dataset.reduceMotion = String(settings.reduceMotion)
    root.style.setProperty('--wallpaper-accent', settings.wallpaperAccent)
  })

  $effect(() => {
    let off = () => {}
    let disposed = false
    // Once for the app's lifetime: init writes the lock state it would
    // otherwise track.
    void untrack(() => appLock.init()).then((stop) => (disposed ? stop() : (off = stop)))
    return () => {
      disposed = true
      off()
    }
  })
</script>

<!-- Started locked: nothing connects or loads until the first unlock.
     Locked later: the lock only covers the desktop, which keeps running. -->
{#if appLock.ready}
  {#if appLock.everUnlocked}<Desktop />{/if}
  {#if appLock.state.locked}<LockScreen />{/if}
{/if}
