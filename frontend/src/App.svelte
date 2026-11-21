<script lang="ts">
  import { untrack } from 'svelte'
  import { MediaQuery } from 'svelte/reactivity'

  import { setWindowBackground } from './lib/api/runtime'
  import Desktop from './lib/shell/Desktop.svelte'
  import LockScreen from './lib/shell/LockScreen.svelte'
  import { perfCheck } from './lib/shell/perfCheck.svelte'
  import { appLock } from './lib/stores/lock.svelte'
  import { resolveTheme, settings } from './lib/stores/settings.svelte'
  import { mixWallpaper } from './lib/wallpaper'

  const systemDark = new MediaQuery('prefers-color-scheme: dark')

  $effect(() => {
    const root = document.documentElement
    const knobs = perfCheck.probing ?? settings.knobs
    root.dataset.theme = resolveTheme(settings.theme, systemDark.current)
    root.dataset.perfBlur = String(knobs.blur)
    root.dataset.perfOpaque = String(knobs.opaque)
    root.dataset.perfShadow = knobs.shadow
    root.dataset.perfWallpaper = knobs.wallpaper ? 'blobs' : 'flat'
    root.dataset.reduceMotion = String(!knobs.motion)
    const accent = settings.wallpaperAccent
    root.style.setProperty('--wallpaper-accent', accent)
    const style = getComputedStyle(root)
    const base = style.getPropertyValue('--wallpaper-base').trim()
    const mix = Number.parseFloat(style.getPropertyValue('--wallpaper-mix'))
    const flat = mixWallpaper(accent, base, Number.isFinite(mix) ? mix : 100)
    if (flat) root.style.setProperty('--wallpaper-flat', flat)
    setWindowBackground(!knobs.wallpaper && flat ? flat : base)
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
