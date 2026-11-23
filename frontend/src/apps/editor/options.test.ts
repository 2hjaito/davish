import { EditorState } from '@codemirror/state'
import { describe, expect, it } from 'vitest'

import {
  gutterExtension,
  indentExtension,
  indentString,
  TAB_WIDTH,
  whitespaceExtension,
  wrapExtension,
} from './options'
import { indentUnit } from '@codemirror/language'

describe('indentString', () => {
  it.each([
    ['2', '  '],
    ['4', '    '],
    ['tab', '\t'],
  ] as const)('%s', (indent, want) => {
    expect(indentString(indent)).toBe(want)
  })
})

describe('editor extensions', () => {
  it('sets the indent unit and keeps the tab width', () => {
    for (const i of ['2', '4', 'tab'] as const) {
      const state = EditorState.create({ extensions: [indentExtension(i)] })
      expect(state.facet(indentUnit)).toBe(indentString(i))
      expect(state.tabSize).toBe(TAB_WIDTH)
    }
  })

  it('adds nothing when a switch is off, and something when it is on', () => {
    for (const make of [wrapExtension, gutterExtension, whitespaceExtension]) {
      expect(make(false)).toEqual([])
      expect(make(true)).not.toEqual([])
    }
  })
})
