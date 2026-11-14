import { describe, expect, it } from 'vitest'
import { splitRatio, storedRatio } from './split'

describe('splitRatio', () => {
  it('follows the pointer inside the limits', () => {
    expect(splitRatio(500, 1000)).toBe(0.5)
  })

  it('keeps room for the list and the detail', () => {
    expect(splitRatio(10, 1000)).toBe(0.22)
    expect(splitRatio(990, 1000)).toBe(0.72)
  })

  it('gives up when there is no room to split', () => {
    expect(splitRatio(100, 400)).toBeNull()
    expect(splitRatio(Number.NaN, 1000)).toBeNull()
  })
})

describe('storedRatio', () => {
  it('drops junk', () => {
    expect(storedRatio(0.4)).toBe(0.4)
    expect(storedRatio('0.4')).toBeNull()
    expect(storedRatio(2)).toBeNull()
    expect(storedRatio(null)).toBeNull()
  })
})
