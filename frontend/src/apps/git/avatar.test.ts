import { describe, expect, it } from 'vitest'
import {
  AVATAR_COLORS,
  avatarColor,
  avatarUrls,
  githubAvatar,
  gravatar,
  initials,
  markFailed,
} from './avatar'

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

describe('githubAvatar', () => {
  it('reads the account id of a noreply address', () => {
    expect(githubAvatar('49699333+dependabot[bot]@users.noreply.github.com')).toBe(
      'https://avatars.githubusercontent.com/u/49699333?s=40&v=4',
    )
  })

  it('falls back to the login of an old noreply address', () => {
    expect(githubAvatar('octocat@users.noreply.github.com')).toBe(
      'https://github.com/octocat.png?size=40',
    )
  })

  it('ignores other addresses', () => {
    expect(githubAvatar('dev@example.com')).toBeNull()
  })
})

describe('gravatar', () => {
  it('hashes the trimmed lowercase email', async () => {
    // sha256("test@example.com")
    expect(await gravatar('  Test@Example.com ')).toBe(
      'https://gravatar.com/avatar/973dfe463ec85785f5f95af5ba3906eedb2d931c24e69824a89ea65dba4e813b?s=40&d=404',
    )
  })

  it('skips a value that is not an email', async () => {
    expect(await gravatar('')).toBeNull()
  })
})

describe('avatarUrls', () => {
  it('tries GitHub first and drops failed pictures', async () => {
    const email = '1+a@users.noreply.github.com'
    const urls = await avatarUrls(email)
    expect(urls[0]).toBe('https://avatars.githubusercontent.com/u/1?s=40&v=4')
    expect(urls).toHaveLength(2)
    markFailed(urls[0] ?? '')
    expect(await avatarUrls(email)).toEqual(urls.slice(1))
  })
})
