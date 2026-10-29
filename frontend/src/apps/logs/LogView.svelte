<script lang="ts">
  import { lineLevel } from '../../lib/logLevel'
  import { findPaths } from '../../lib/logPaths'
  import { t } from '../../lib/i18n'

  interface Props {
    lines: readonly string[]
    /** Follows new lines while the view is at the bottom. */
    follow: boolean
    onopen: (path: string, line?: number) => void
  }

  let { lines, follow, onopen }: Props = $props()

  // Must match .line's height in CSS: rows are absolutely placed from it.
  const ROW_H = 18
  const OVERSCAN = 20

  let scroller = $state<HTMLDivElement>()
  let scrollTop = $state(0)
  let viewport = $state(0)
  /** The view sits at the bottom, so new lines scroll it along. */
  let atBottom = $state(true)

  const start = $derived(Math.max(0, Math.floor(scrollTop / ROW_H) - OVERSCAN))
  const end = $derived(Math.min(lines.length, Math.ceil((scrollTop + viewport) / ROW_H) + OVERSCAN))
  const rows = $derived(lines.slice(start, end))

  function onscroll() {
    if (!scroller) return
    scrollTop = scroller.scrollTop
    atBottom = scroller.scrollHeight - scroller.scrollTop - scroller.clientHeight < ROW_H
  }

  $effect(() => {
    void lines.length
    if (follow && atBottom && scroller) {
      scroller.scrollTop = scroller.scrollHeight
      scrollTop = scroller.scrollTop
    }
  })
</script>

<!-- Server output is shown as text only (docs/SECURITY.md); paths become
     buttons that open the Editor, nothing else is interpreted. -->
<!-- Focusable so the keyboard can scroll it, as a scrollable region should be. -->
<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
<div
  class="scroller"
  role="log"
  aria-label={t.logs.view}
  tabindex="0"
  bind:this={scroller}
  bind:clientHeight={viewport}
  {onscroll}
>
  <div class="rows" style:height="{lines.length * ROW_H}px">
    {#each rows as line, i (start + i)}
      <div class="line {lineLevel(line)}" style:translate="0 {(start + i) * ROW_H}px">
        {#each findPaths(line) as seg, j (j)}
          {#if seg.path}
            <button
              type="button"
              class="path"
              title={t.logs.openPath(seg.path, seg.line)}
              onclick={() => onopen(seg.path!, seg.line)}>{seg.text}</button
            >
          {:else}{seg.text}{/if}
        {/each}
      </div>
    {/each}
  </div>
</div>

<style>
  .scroller {
    flex: 1;
    min-height: 0;
    overflow: auto;
    padding: 6px 0 10px;
    outline: none;
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    color: var(--fg-on-dark);
    user-select: text;
  }

  .rows {
    position: relative;
    min-width: 100%;
    width: max-content;
  }

  .line {
    position: absolute;
    top: 0;
    left: 0;
    height: 18px;
    line-height: 18px;
    padding: 0 14px;
    white-space: pre;
  }

  .line.error {
    color: var(--danger-on-dark);
  }

  .line.warn {
    color: var(--warn-on-dark);
  }

  .path {
    padding: 0;
    border: 0;
    background: none;
    font: inherit;
    color: var(--term-path);
    text-decoration: underline;
    text-decoration-color: color-mix(in srgb, currentColor 40%, transparent);
    text-underline-offset: 2px;
    cursor: pointer;
  }

  .path:hover {
    text-decoration-color: currentColor;
  }

  .scroller::-webkit-scrollbar {
    width: 6px;
    height: 6px;
  }

  .scroller::-webkit-scrollbar-thumb {
    border-radius: 3px;
    background: var(--term-scrollbar);
  }

  .scroller::-webkit-scrollbar-thumb:hover {
    background: var(--term-scrollbar-hover);
  }

  .scroller::-webkit-scrollbar-corner {
    background: none;
  }
</style>
