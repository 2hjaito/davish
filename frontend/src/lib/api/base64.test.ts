import { describe, expect, it } from 'vitest'

import { decodeBase64, encodeBase64 } from './base64'

describe('base64', () => {
  it('keeps raw bytes, including ones that are not valid UTF-8', () => {
    // "hé" in UTF-8 is 68 c3 a9; ff is never valid UTF-8.
    const bytes = new Uint8Array([0x68, 0xc3, 0xa9, 0xff])
    expect(encodeBase64(bytes)).toBe('aMOp/w==')
    expect(Array.from(decodeBase64('aMOp/w=='))).toEqual(Array.from(bytes))
  })

  it('round-trips data larger than one chunk', () => {
    const bytes = new Uint8Array(100_000).map((_, i) => i % 256)
    expect(decodeBase64(encodeBase64(bytes))).toEqual(bytes)
  })

  it('handles empty input', () => {
    expect(encodeBase64(new Uint8Array())).toBe('')
    expect(decodeBase64('')).toEqual(new Uint8Array())
  })
})
