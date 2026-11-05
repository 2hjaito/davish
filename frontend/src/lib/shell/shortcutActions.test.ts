import { describe, expect, it } from 'vitest'

import type { ShortcutTarget } from '../api/shortcuts'
import { hasChanged, needsConfirm } from './shortcutActions'

const target = (hash: string): ShortcutTarget => ({
  name: 'deploy.sh',
  kind: 'script',
  hash,
  command: './deploy.sh',
})

describe('needsConfirm', () => {
  it('asks the first time', () => {
    expect(needsConfirm({}, target('aa'))).toBe(true)
    expect(hasChanged({}, target('aa'))).toBe(false)
  })

  it('runs a trusted script while its content stays', () => {
    expect(needsConfirm({ trusted: 'aa' }, target('aa'))).toBe(false)
    expect(hasChanged({ trusted: 'aa' }, target('aa'))).toBe(false)
  })

  it('asks again, as changed, when the content differs', () => {
    expect(needsConfirm({ trusted: 'aa' }, target('bb'))).toBe(true)
    expect(hasChanged({ trusted: 'aa' }, target('bb'))).toBe(true)
  })

  it('always asks when the content could not be read', () => {
    expect(needsConfirm({ trusted: '' }, target(''))).toBe(true)
    expect(needsConfirm({}, target(''))).toBe(true)
  })
})
