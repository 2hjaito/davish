import { describe, expect, it } from 'vitest'

import { toUserError } from './errors'

describe('toUserError', () => {
  it('maps a known code', () => {
    expect(toUserError({ code: 'CMD_NEEDS_ROOT', message: 'x' }).code).toBe('CMD_NEEDS_ROOT')
  })

  it.each([
    ['unknown code', { code: 'NOPE' }],
    ['plain string', 'boom'],
    ['null', null],
  ])('falls back to INTERNAL for %s', (_label, input) => {
    expect(toUserError(input).code).toBe('INTERNAL')
  })
})
