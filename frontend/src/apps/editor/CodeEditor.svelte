<script lang="ts">
  import { closeBrackets, closeBracketsKeymap } from '@codemirror/autocomplete'
  import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands'
  import { bracketMatching, indentOnInput } from '@codemirror/language'
  import { highlightSelectionMatches, searchKeymap } from '@codemirror/search'
  import { Compartment, EditorState } from '@codemirror/state'
  import {
    drawSelection,
    EditorView,
    highlightActiveLine,
    highlightActiveLineGutter,
    highlightSpecialChars,
    keymap,
    lineNumbers,
  } from '@codemirror/view'
  import { onMount, untrack } from 'svelte'

  import { t } from '../../lib/i18n/vi'
  import type { LanguageId } from './languages'
  import { highlighting, languageSupport } from './syntax'

  interface Props {
    /** The text it starts with; later edits stay inside the editor. */
    doc: string
    language: LanguageId
    readOnly?: boolean
    /** Line to put the cursor on when opening, 1-based. */
    line?: number
    /** Called on every edit and cursor move. */
    onchange?: (info: { text: () => string; line: number; col: number; edited: boolean }) => void
    /** ⌘S / Ctrl+S. */
    onsave?: () => void
  }

  let { doc, language, readOnly = false, line, onchange, onsave }: Props = $props()

  let host: HTMLDivElement
  let view: EditorView | undefined
  const lang = new Compartment()
  const ro = new Compartment()

  // Colors and sizes from tokens.css; CodeMirror injects this as a scoped style.
  const theme = EditorView.theme({
    '&': {
      height: '100%',
      fontSize: 'var(--text-sm)',
      color: 'var(--fg)',
      backgroundColor: 'transparent',
    },
    '.cm-scroller': { fontFamily: 'var(--font-mono)', lineHeight: '1.55' },
    '.cm-content': { caretColor: 'var(--accent)', padding: '6px 0' },
    '&.cm-focused': { outline: 'none' },
    '.cm-gutters': {
      backgroundColor: 'transparent',
      color: 'var(--editor-gutter-fg)',
      border: 'none',
      paddingLeft: '6px',
    },
    '.cm-activeLine': { backgroundColor: 'var(--editor-active-line)' },
    '.cm-activeLineGutter': { backgroundColor: 'transparent', color: 'var(--fg-muted)' },
    '&.cm-focused .cm-selectionBackground, .cm-selectionBackground, ::selection': {
      backgroundColor: 'var(--editor-selection)',
    },
    '.cm-selectionMatch, .cm-searchMatch': { backgroundColor: 'var(--editor-match)' },
    '.cm-cursor': { borderLeftColor: 'var(--accent)' },
    '.cm-panels': {
      backgroundColor: 'var(--content-bg)',
      color: 'var(--fg)',
      borderBottom: '1px solid var(--divider-soft)',
    },
    '.cm-panel.cm-search': { fontFamily: 'var(--font-ui)', fontSize: 'var(--text-sm)' },
    '.cm-textfield, .cm-button': {
      borderRadius: 'var(--control-radius)',
      border: '1px solid var(--divider)',
      backgroundImage: 'none',
      backgroundColor: 'var(--content-bg)',
    },
  })

  function report(v: EditorView, edited: boolean) {
    const head = v.state.selection.main.head
    const l = v.state.doc.lineAt(head)
    onchange?.({
      text: () => v.state.doc.toString(),
      line: l.number,
      col: head - l.from + 1,
      edited,
    })
  }

  onMount(() => {
    const v = new EditorView({
      parent: host,
      state: EditorState.create({
        doc: untrack(() => doc),
        extensions: [
          EditorState.phrases.of(t.editor.phrases),
          lineNumbers(),
          highlightActiveLineGutter(),
          highlightSpecialChars(),
          history(),
          drawSelection(),
          indentOnInput(),
          bracketMatching(),
          closeBrackets(),
          highlightActiveLine(),
          highlightSelectionMatches(),
          // Before the defaults, so the browser never sees ⌘S.
          keymap.of([{ key: 'Mod-s', preventDefault: true, run: () => (onsave?.(), true) }]),
          keymap.of([
            ...closeBracketsKeymap,
            ...defaultKeymap,
            ...searchKeymap,
            ...historyKeymap,
            indentWithTab,
          ]),
          highlighting,
          theme,
          lang.of(languageSupport(untrack(() => language))),
          ro.of(EditorState.readOnly.of(untrack(() => readOnly))),
          EditorView.updateListener.of((u) => {
            if (u.docChanged || u.selectionSet) report(u.view, u.docChanged)
          }),
        ],
      }),
    })
    view = v
    const start = untrack(() => line)
    if (start && start <= v.state.doc.lines) {
      const at = v.state.doc.line(start).from
      v.dispatch({ selection: { anchor: at }, scrollIntoView: true })
    }
    report(v, false)
    v.focus()
    return () => v.destroy()
  })

  $effect(() => {
    const ext = languageSupport(language)
    view?.dispatch({ effects: lang.reconfigure(ext) })
  })

  $effect(() => {
    const value = readOnly
    view?.dispatch({ effects: ro.reconfigure(EditorState.readOnly.of(value)) })
  })

  /** Moves the cursor to a 1-based line and shows it. */
  export function goToLine(n: number) {
    if (!view || n < 1 || n > view.state.doc.lines) return
    view.dispatch({ selection: { anchor: view.state.doc.line(n).from }, scrollIntoView: true })
    view.focus()
  }

  /** The current text, edits included. */
  export function getText(): string {
    return view?.state.doc.toString() ?? ''
  }

  export function focus() {
    view?.focus()
  }
</script>

<div class="code" bind:this={host}></div>

<style>
  .code {
    flex: 1;
    min-height: 0;
    overflow: hidden;
  }

  .code :global(.cm-editor) {
    height: 100%;
  }
</style>
