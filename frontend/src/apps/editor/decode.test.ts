import { describe, expect, it } from 'vitest'

import { decodeText } from './decode'

const bytes = (...b: number[]) => new Uint8Array(b)

describe('decodeText', () => {
  it('reads UTF-8, Vietnamese included', () => {
    expect(decodeText(new TextEncoder().encode('máy chủ = "tốt"\n'))).toBe('máy chủ = "tốt"\n')
  })

  it('keeps a byte order mark', () => {
    expect(decodeText(bytes(0xef, 0xbb, 0xbf, 0x61))).toBe('﻿a')
  })

  it('refuses binary files and other encodings', () => {
    expect(decodeText(bytes(0x7f, 0x45, 0x4c, 0x46, 0x00, 0x01))).toBeNull() // ELF
    expect(decodeText(bytes(0x63, 0x61, 0x66, 0xe9))).toBeNull() // "café" in Latin-1
  })

  it('reads an empty file', () => {
    expect(decodeText(bytes())).toBe('')
  })
})
