<script lang="ts">
  import { lineLevel } from '../../lib/logLevel'
  import { findPaths } from '../../lib/logPaths'
  import { t } from '../../lib/i18n'
  import { type Point, sliceLines } from './selection'

  interface Props {
    lines: readonly string[]
    /** Follows new lines while the view is at the bottom. */
    follow: boolean
    onopen: (path: string, line?: number) => void
  }

  let { lines: incoming, follow, onopen }: Props = $props()

  // While text is selected in the view, what it shows holds still: new lines
  // would scroll it or, past the buffer's cap, shift every row under the
  // selection. They show up once the selection is gone.
  let frozen = $state.raw<readonly string[] | null>(null)
  const lines = $derived(frozen ?? incoming)

  function selectionInView(): Selection | null {
    const sel = document.getSelection()
    if (!sel || sel.isCollapsed || !scroller || !sel.anchorNode) return null
    return scroller.contains(sel.anchorNode) ? sel : null
  }

  function onselectionchange() {
    if (selectionInView()) frozen ??= incoming
    else frozen = null
  }

  /** Where a DOM point falls in the log, from its row's data-row. */
  function point(node: Node, offset: number): Point | null {
    const el = node instanceof Element ? node : node.parentElement
    const row = el?.closest<HTMLElement>('[data-row]')
    if (!row) return null
    const r = document.createRange()
    r.setStart(row, 0)
    r.setEnd(node, offset)
    return { row: Number(row.dataset.row), offset: r.toString().length }
  }

  // ⌘C copies the lines themselves, breaks included.
  function oncopy(e: ClipboardEvent) {
    const sel = selectionInView()
    if (!sel?.anchorNode || !sel.focusNode || !e.clipboardData) return
    const a = point(sel.anchorNode, sel.anchorOffset)
    const b = point(sel.focusNode, sel.focusOffset)
    if (!a || !b) return
    e.preventDefault()
    e.clipboardData.setData('text/plain', sliceLines(lines, a, b))
  }

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
    if (follow && atBottom && scroller && !frozen) {
      scroller.scrollTop = scroller.scrollHeight
      scrollTop = scroller.scrollTop
    }
  })
</script>

<svelte:document {onselectionchange} />

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
  {oncopy}
>
  <div class="rows" style:height="{lines.length * ROW_H}px">
    {#each rows as line, i (start + i)}
      <div
        class="line {lineLevel(line)}"
        data-row={start + i}
        style:translate="0 {(start + i) * ROW_H}px"
      >
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
    /* body turns selection off for the whole UI; WebKit (macOS, Linux) only
       reads the prefixed property, so both are needed to turn it back on. */
    -webkit-user-select: text;
    user-select: text;
    cursor: text;
  }

  .scroller ::selection {
    background: color-mix(in srgb, var(--accent) 45%, transparent);
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
    -webkit-user-select: text;
    user-select: text;
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
