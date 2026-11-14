import { describe, expect, it } from 'vitest'
import { AVATAR_COLORS, avatarColor, initials } from './avatar'

describe('initials', () => {
  it('takes the first and last word', () => {
    expect(initials('Tran Huu Dang')).toBe('TD')
    expect(initials('Sofia Novak')).toBe('SN')
  })

  it('takes two letters of a single word', () => {
    expect(initials('dependabot')).toBe('DE')
    expect(initials('x')).toBe('X')
  })

  it('splits handles on punctuation', () => {
    expect(initials('john.doe')).toBe('JD')
    expect(initials('github-actions[bot]')).toBe('GA')
  })

  it('keeps accented letters whole', () => {
    expect(initials('Đặng Ánh')).toBe('ĐÁ')
  })

  it('falls back for an empty name', () => {
    expect(initials('   ')).toBe('?')
  })
})

describe('avatarColor', () => {
  it('is stable and in range', () => {
    const c = avatarColor('Maya Chen')
    expect(avatarColor(' maya chen ')).toBe(c)
    expect(c).toBeGreaterThanOrEqual(0)
    expect(c).toBeLessThan(AVATAR_COLORS)
  })
})
