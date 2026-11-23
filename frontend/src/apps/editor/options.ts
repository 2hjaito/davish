// What the Editor's settings mean to CodeMirror, kept apart from the
// component so it can be tested.
import { indentUnit } from '@codemirror/language'
import { EditorState, type Extension } from '@codemirror/state'
import {
  EditorView,
  highlightActiveLineGutter,
  highlightWhitespace,
  lineNumbers,
} from '@codemirror/view'

import type { EditorIndent } from '../../lib/stores/settings.svelte'

/** What one level of indentation is made of. */
export function indentString(indent: EditorIndent): string {
  return indent === 'tab' ? '\t' : ' '.repeat(Number(indent))
}

/**
 * How wide a tab character is drawn. Fixed, as it was before it could be
 * chosen: a file with tabs reads the same whatever the indent is.
 */
export const TAB_WIDTH = 4

/** The indent the Tab key and auto-indent use. */
export function indentExtension(indent: EditorIndent): Extension {
  return [indentUnit.of(indentString(indent)), EditorState.tabSize.of(TAB_WIDTH)]
}

export const wrapExtension = (on: boolean): Extension => (on ? EditorView.lineWrapping : [])

export const gutterExtension = (on: boolean): Extension =>
  on ? [lineNumbers(), highlightActiveLineGutter()] : []

export const whitespaceExtension = (on: boolean): Extension => (on ? highlightWhitespace() : [])
