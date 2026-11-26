<script lang="ts">
  import { untrack } from 'svelte'
  import { MediaQuery } from 'svelte/reactivity'

  import { startBench } from './lib/api/bench'
  import { setWindowBackground } from './lib/api/runtime'
  import Desktop from './lib/shell/Desktop.svelte'
  import { wm } from './lib/wm/windowManager.svelte'
  import LockScreen from './lib/shell/LockScreen.svelte'
  import { perfCheck } from './lib/shell/perfCheck.svelte'
  import { appLock } from './lib/stores/lock.svelte'
  import { resolveTheme, settings } from './lib/stores/settings.svelte'
  import { wallpaperImage } from './lib/stores/wallpaperImage.svelte'
  import { mixWallpaper, wallpaperMode } from './lib/wallpaper'

  const systemDark = new MediaQuery('prefers-color-scheme: dark')

  $effect(() => {
    const root = document.documentElement
    const knobs = perfCheck.probing ?? settings.knobs
    root.dataset.theme = resolveTheme(settings.theme, systemDark.current)
    // While a window is dragged in the simple mode, the light look is painted.
    const lite = wm.dragLite
    root.dataset.perfBlur = String(knobs.blur && !lite)
    root.dataset.perfOpaque = String(knobs.opaque || lite)
    root.dataset.perfShadow = lite ? 'short' : knobs.shadow
    // The picture is read from disk only when the preset lets it show.
    if (knobs.wallpaperImage) void wallpaperImage.ensure()
    const mode = wallpaperMode(knobs, wallpaperImage.has)
    root.dataset.perfWallpaper = mode
    root.dataset.reduceMotion = String(!knobs.motion)
    const accent = settings.wallpaperAccent
    root.style.setProperty('--wallpaper-accent', accent)
    const style = getComputedStyle(root)
    const base = style.getPropertyValue('--wallpaper-base').trim()
    const mix = Number.parseFloat(style.getPropertyValue('--wallpaper-mix'))
    const flat = mixWallpaper(accent, base, Number.isFinite(mix) ? mix : 100)
    if (flat) root.style.setProperty('--wallpaper-flat', flat)
    setWindowBackground(mode === 'flat' && flat ? flat : base)
    if (!root.dataset.wallpaperReady)
      requestAnimationFrame(() => (root.dataset.wallpaperReady = 'true'))
  })

  // Only when scripts/bench.sh --render started this process.
  $effect(() => {
    void startBench()
  })

  // WebKit can keep the menu bar's, Dock's and desktop icons' blur of the old wallpaper until
  // something else repaints them. Once the color has glided (tokens.css),
  // nudge their opacity by a hair, which makes it blur the new one.
  let nudge = 1
  $effect(() => {
    void settings.wallpaperAccent
    const timer = setTimeout(() => {
      nudge = nudge === 1 ? 0.999 : 1
      document.documentElement.style.setProperty('--backdrop-nudge', String(nudge))
    }, 600)
    return () => clearTimeout(timer)
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
