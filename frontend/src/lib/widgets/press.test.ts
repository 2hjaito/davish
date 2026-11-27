import { describe, expect, it } from 'vitest'

import { endPress, holdPress, movePress, startPress } from './press'

describe('movePress', () => {
  it('is not a drag under 5 px, and is one at 5', () => {
    const p = startPress(100, 100)
    expect(movePress(p, 104, 100).dragging).toBe(false)
    expect(movePress(p, 103, 104).dragging).toBe(true) // hypot = 5
  })

  it('stays a drag when the pointer comes back', () => {
    const p = movePress(startPress(0, 0), 20, 0)
    expect(movePress(p, 0, 0).dragging).toBe(true)
  })
})

describe('holdPress', () => {
  it('holds a press that has not moved', () => {
    expect(holdPress(startPress(0, 0)).held).toBe(true)
  })

  it('a small wobble is still a hold', () => {
    expect(holdPress(movePress(startPress(0, 0), 3, 2)).held).toBe(true)
  })

  it('does not hold a drag', () => {
    expect(holdPress(movePress(startPress(0, 0), 10, 0)).held).toBe(false)
  })
})

describe('endPress', () => {
  it('a short press is a click', () => {
    expect(endPress(startPress(0, 0))).toBe('click')
  })

  it('a hold swallows the click', () => {
    expect(endPress(holdPress(startPress(0, 0)))).toBe('swallow')
  })

  it('a drag drops', () => {
    expect(endPress(movePress(startPress(0, 0), 10, 0))).toBe('drop')
  })

  it('a held press that then moves is a drag', () => {
    const p = movePress(holdPress(startPress(0, 0)), 10, 0)
    expect(p.held).toBe(true)
    expect(endPress(p)).toBe('drop')
  })
})
