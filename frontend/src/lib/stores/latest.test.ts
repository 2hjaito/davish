import { describe, expect, it } from 'vitest'

import { supersedes } from './latest'

describe('supersedes', () => {
  it('keeps only the newest attempt current', () => {
    const attempt = supersedes()
    const first = attempt()
    expect(first()).toBe(true)
    const second = attempt()
    expect(first()).toBe(false)
    expect(second()).toBe(true)
  })
})
