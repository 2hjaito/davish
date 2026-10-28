<script lang="ts">
  import '@xterm/xterm/css/xterm.css'

  import { FitAddon } from '@xterm/addon-fit'
  import { WebglAddon } from '@xterm/addon-webgl'
  import { type ITheme, Terminal } from '@xterm/xterm'
  import { untrack } from 'svelte'

  import { isMac } from '../../lib/shell/platform'
  import { settings } from '../../lib/stores/settings.svelte'

  interface Props {
    /** Whether this terminal is the visible tab; hidden ones keep their buffer. */
    active: boolean
    onready: (term: Terminal) => void
  }

  let { active, onready }: Props = $props()

  const LINE_HEIGHT = 1.35

  let host = $state<HTMLDivElement>()
  let term: Terminal | undefined
  let fit: FitAddon | undefined
  let webgl: WebglAddon | undefined

  function token(name: string): string {
    return getComputedStyle(document.documentElement).getPropertyValue(name).trim()
  }

  function theme(solid: boolean): ITheme {
    return {
      background: solid ? token('--term-bg-solid') : '#00000000',
      foreground: token('--fg-on-dark'),
      cursor: token('--fg-on-dark'),
      cursorAccent: token('--app-terminal'),
      selectionBackground: token('--term-selection'),
      scrollbarSliderBackground: token('--term-scrollbar'),
      scrollbarSliderHoverBackground: token('--term-scrollbar-hover'),
      scrollbarSliderActiveBackground: token('--term-scrollbar-hover'),
      green: token('--term-user'),
      brightGreen: token('--term-user'),
      blue: token('--term-path'),
      brightBlue: token('--term-path'),
      yellow: token('--term-warn'),
      brightYellow: token('--term-warn'),
    }
  }

  // Ctrl+Shift+C/V on Windows and Linux, where plain Ctrl+C must reach the shell.
  // macOS uses ⌘C/⌘V through the native Edit menu.
  function clipboardKeys(e: KeyboardEvent): boolean {
    if (isMac || e.type !== 'keydown' || !e.ctrlKey || !e.shiftKey) return true
    const key = e.key.toLowerCase()
    if (key === 'c' && term?.hasSelection()) {
      void navigator.clipboard.writeText(term.getSelection())
      return false
    }
    if (key === 'v') {
      void navigator.clipboard.readText().then((text) => term?.paste(text))
      return false
    }
    return true
  }

  /**
   * The block cursor fills the whole row, which lineHeight makes taller than the
   * text. Keep the row height but paint only the glyph's own height (see CSS).
   * Only the DOM renderer has these rows; WebGL paints the full-height cursor.
   */
  function fitCursor(t: Terminal) {
    const row = t.element?.querySelector<HTMLElement>('.xterm-rows > div')
    if (!row || !host) return
    const glyph = row.offsetHeight / LINE_HEIGHT
    host.style.setProperty('--cursor-h', `${glyph}px`)
    host.style.setProperty('--cursor-inset', `${(row.offsetHeight - glyph) / 2}px`)
  }

  /**
   * The DOM renderer draws on the transparent glass. WebGL (Settings → GPU)
   * bakes each cell's background into its glyph atlas, which misplaces any
   * coloured cell (selection, vim's status line, powerline prompts) on a
   * transparent terminal, so it draws on a solid background instead. Without
   * WebGL2, or once the context is lost, the terminal goes back to the DOM.
   */
  function setRenderer(t: Terminal, gpu: boolean) {
    if (gpu === Boolean(webgl)) return
    if (gpu) {
      try {
        const w = new WebglAddon()
        w.onContextLoss(() => setRenderer(t, false))
        t.options.allowTransparency = false
        t.options.theme = theme(true)
        t.loadAddon(w)
        webgl = w
        return
      } catch {
        // No WebGL2: stay on the DOM renderer below.
      }
    }
    webgl?.dispose()
    webgl = undefined
    t.options.allowTransparency = true
    t.options.theme = theme(false)
    fitCursor(t)
  }

  $effect(() => {
    if (!host) return
    const t = new Terminal({
      allowTransparency: true,
      cursorBlink: true,
      fontFamily: untrack(() => settings.monoFamily),
      // Read once here; the effect below applies later changes in place.
      fontSize: untrack(() => settings.terminalFontSize),
      lineHeight: LINE_HEIGHT,
      scrollback: 5000,
      macOptionIsMeta: true,
      theme: theme(false),
    })
    const f = new FitAddon()
    t.loadAddon(f)
    t.attachCustomKeyEventHandler(clipboardKeys)
    t.open(host)
    setRenderer(
      t,
      untrack(() => settings.terminalGpu),
    )
    term = t
    fit = f

    // Measure after the bundled font is ready, or cell widths come out wrong.
    void document.fonts.ready.then(() => {
      if (term !== t) return
      f.fit()
      fitCursor(t)
      onready(t)
    })

    const ro = new ResizeObserver(() => {
      if (host?.offsetParent) f.fit()
    })
    ro.observe(host)

    return () => {
      ro.disconnect()
      t.dispose()
      term = undefined
      fit = undefined
      webgl = undefined
    }
  })

  $effect(() => {
    const gpu = settings.terminalGpu
    if (term) setRenderer(term, gpu)
  })

  $effect(() => {
    const family = settings.monoFamily
    if (!term || term.options.fontFamily === family) return
    term.options.fontFamily = family
    fit?.fit()
    fitCursor(term)
  })

  $effect(() => {
    const size = settings.terminalFontSize
    if (!term || term.options.fontSize === size) return
    term.options.fontSize = size
    fit?.fit()
    fitCursor(term)
  })

  $effect(() => {
    if (active && term) {
      fit?.fit()
      term.focus()
    }
  })

  /** Focuses the terminal so typing goes to it. */
  export function focus() {
    term?.focus()
  }
</script>

<div class="host" class:hidden={!active} bind:this={host}></div>

<style>
  .host {
    position: absolute;
    inset: 10px 6px 6px 16px;
  }

  .hidden {
    visibility: hidden;
  }

  .host :global(.xterm) {
    height: 100%;
  }

  .host :global(.xterm-rows .xterm-cursor.xterm-cursor-block) {
    line-height: var(--cursor-h);
    padding-block: var(--cursor-inset);
    background-clip: content-box;
  }

  /*
   * xterm 6 scrolls with its own scrollbar, yet the viewport keeps
   * overflow-y: scroll. When macOS always shows scroll bars (a mouse is
   * attached), WebKit paints an empty light track there; hide it.
   */
  .host :global(.xterm-viewport) {
    background: transparent !important;
    scrollbar-width: none;
  }

  .host :global(.xterm-viewport::-webkit-scrollbar) {
    display: none;
  }

  /* A thin macOS-style overlay slider in place of xterm's 14px VS Code bar. */
  .host :global(.xterm-scrollable-element > .scrollbar.vertical > .slider) {
    left: 6px !important;
    width: 6px !important;
    border-radius: 3px;
  }
</style>
