import { describe, expect, it } from 'vitest'

import { decodeOutput } from './terminal'

describe('decodeOutput', () => {
  it('keeps raw bytes, including ones that are not valid UTF-8', () => {
    // "hé" in UTF-8 is 68 c3 a9; ff is never valid UTF-8.
    expect(Array.from(decodeOutput('aMOp/w=='))).toEqual([0x68, 0xc3, 0xa9, 0xff])
  })
})
