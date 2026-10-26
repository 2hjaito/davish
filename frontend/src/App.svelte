<script lang="ts">
  import Desktop from './lib/shell/Desktop.svelte'
  import { settings } from './lib/stores/settings.svelte'

  $effect(() => {
    const root = document.documentElement
    root.dataset.reduceGlass = String(settings.reduceGlass)
    root.dataset.reduceMotion = String(settings.reduceMotion)
    root.style.setProperty('--wallpaper-accent', settings.wallpaperAccent)
  })
</script>

<Desktop />
