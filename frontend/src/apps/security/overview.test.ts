import { describe, expect, it } from 'vitest'
import { hourLabel, shares } from './overview'

describe('hourLabel', () => {
  it('keeps month, day and hour', () => {
    expect(hourLabel('2026-10-28T05')).toBe('10-28 05h')
  })
  it('leaves anything else as it is', () => {
    expect(hourLabel('Oct 28')).toBe('Oct 28')
  })
})

describe('shares', () => {
  it('measures against the largest', () => {
    expect(shares([50, 25, 0])).toEqual([100, 50, 0])
  })
  it('keeps a sliver for tiny counts', () => {
    expect(shares([1000, 1])).toEqual([100, 2])
  })
  it('is all zero without counts', () => {
    expect(shares([])).toEqual([])
    expect(shares([0, 0])).toEqual([0, 0])
  })
})
