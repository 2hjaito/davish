<script lang="ts" module>
  export type LineKind = 'add' | 'del' | 'hunk' | 'meta' | 'ctx'

  /** How a line of a unified diff is drawn. */
  export function lineKind(line: string): LineKind {
    if (line.startsWith('+++ ') || line.startsWith('--- ')) return 'meta'
    if (line.startsWith('+')) return 'add'
    if (line.startsWith('-')) return 'del'
    if (line.startsWith('@@')) return 'hunk'
    if (line.startsWith(' ') || line === '') return 'ctx'
    return 'meta'
  }
</script>

<script lang="ts">
  import { t } from '../../lib/i18n/vi'

  // Text of the server, so plain text only: never {@html} (docs/SECURITY.md).
  interface Props {
    text: string
    truncated: boolean
  }

  let { text, truncated }: Props = $props()

  /** More lines than this slow the window down; the rest is not drawn. */
  const MAX_LINES = 5000

  const all = $derived(text.replace(/\n$/, '').split('\n'))
  const lines = $derived(all.slice(0, MAX_LINES).map((l) => ({ text: l, kind: lineKind(l) })))
</script>

{#if text.trim() === ''}
  <p class="note">{t.git.diffEmpty}</p>
{:else}
  {#if truncated}<p class="note">{t.git.diffTruncated}</p>{/if}
  {#if all.length > MAX_LINES}<p class="note">{t.git.diffLines(MAX_LINES)}</p>{/if}
  <!-- One block per line; a blank line keeps its height with a space. -->
  <pre class="diff">{#each lines as l, i (i)}<span class={l.kind}>{l.text || ' '}</span
      >{/each}</pre>
{/if}

<style>
  .diff {
    flex: 1;
    min-height: 0;
    margin: 0;
    padding: 8px 0;
    overflow: auto;
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    line-height: 1.55;
    color: var(--fg);
    user-select: text;
  }

  .diff span {
    display: block;
    min-width: max-content;
    padding: 0 14px;
  }

  .add {
    background: var(--diff-added);
  }

  .del {
    background: var(--diff-removed);
  }

  .hunk {
    color: var(--accent);
  }

  .meta {
    color: var(--fg-muted);
  }

  .note {
    margin: 8px 14px 0;
    font-size: var(--text-xs);
    color: var(--warn);
  }
</style>
