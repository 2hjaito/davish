// CodeMirror language support and colors for each LanguageId.
import { json } from '@codemirror/lang-json'
import { markdown } from '@codemirror/lang-markdown'
import { yaml } from '@codemirror/lang-yaml'
import { HighlightStyle, StreamLanguage, syntaxHighlighting } from '@codemirror/language'
import { nginx } from '@codemirror/legacy-modes/mode/nginx'
import { properties } from '@codemirror/legacy-modes/mode/properties'
import { shell } from '@codemirror/legacy-modes/mode/shell'
import type { Extension } from '@codemirror/state'
import { tags } from '@lezer/highlight'

import type { LanguageId } from './languages'

export function languageSupport(id: LanguageId): Extension {
  switch (id) {
    case 'json':
      return json()
    case 'yaml':
      return yaml()
    case 'markdown':
      return markdown()
    case 'nginx':
      return StreamLanguage.define(nginx)
    case 'shell':
      return StreamLanguage.define(shell)
    // .env and most config files are "key = value" with # comments.
    case 'env':
    case 'conf':
      return StreamLanguage.define(properties)
    case 'plain':
      return []
  }
}

// Colors come from tokens.css so the editor follows the design system.
const style = HighlightStyle.define([
  {
    tag: [tags.comment, tags.lineComment, tags.blockComment],
    color: 'var(--code-comment)',
    fontStyle: 'italic',
  },
  {
    tag: [tags.keyword, tags.controlKeyword, tags.operatorKeyword, tags.definitionKeyword],
    color: 'var(--code-keyword)',
  },
  { tag: [tags.string, tags.special(tags.string), tags.regexp], color: 'var(--code-string)' },
  { tag: [tags.number, tags.bool, tags.null, tags.atom], color: 'var(--code-number)' },
  {
    tag: [tags.propertyName, tags.attributeName, tags.definition(tags.propertyName)],
    color: 'var(--code-property)',
  },
  { tag: [tags.variableName, tags.special(tags.variableName)], color: 'var(--code-variable)' },
  { tag: [tags.meta, tags.processingInstruction, tags.typeName], color: 'var(--code-meta)' },
  { tag: tags.heading, fontWeight: '700' },
  { tag: tags.strong, fontWeight: '700' },
  { tag: tags.emphasis, fontStyle: 'italic' },
  { tag: tags.link, color: 'var(--code-property)', textDecoration: 'underline' },
  { tag: tags.invalid, color: 'var(--danger)' },
])

export const highlighting = syntaxHighlighting(style)
