import { describe, expect, it } from 'vitest'

import { lineKind } from './DiffView.svelte'

describe('lineKind', () => {
  it('tells the lines of a unified diff apart', () => {
    expect(lineKind('diff --git a/x b/x')).toBe('meta')
    expect(lineKind('index 3b18e51..a9c4f02 100644')).toBe('meta')
    expect(lineKind('--- a/x')).toBe('meta')
    expect(lineKind('+++ b/x')).toBe('meta')
    expect(lineKind('@@ -1,3 +1,4 @@ fn')).toBe('hunk')
    expect(lineKind('+added')).toBe('add')
    expect(lineKind('-removed')).toBe('del')
    expect(lineKind('---removed dashes')).toBe('del')
    expect(lineKind(' context')).toBe('ctx')
    expect(lineKind('')).toBe('ctx')
    expect(lineKind('\\ No newline at end of file')).toBe('meta')
  })
})
