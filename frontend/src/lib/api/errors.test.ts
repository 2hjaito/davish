import { describe, expect, it } from 'vitest'

import { toUserError } from './errors'

describe('toUserError', () => {
  it('maps a known code', () => {
    expect(toUserError({ code: 'CMD_NEEDS_ROOT', message: 'x' }).code).toBe('CMD_NEEDS_ROOT')
  })

  it("keeps the backend's text as detail", () => {
    const e = toUserError({ code: 'UPDATE_DOWNLOAD_FAILED', message: 'download answered 504' })
    expect(e.detail).toBe('download answered 504')
    expect(e.message).not.toBe('download answered 504')
  })

  it.each([
    ['unknown code', { code: 'NOPE' }],
    ['plain string', 'boom'],
    ['null', null],
  ])('falls back to INTERNAL for %s', (_label, input) => {
    expect(toUserError(input).code).toBe('INTERNAL')
  })
})
