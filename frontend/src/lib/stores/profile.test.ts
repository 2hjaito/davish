import { describe, expect, it } from 'vitest'

import { cleanProfile, MAX_NAME } from './profile.svelte'

describe('cleanProfile', () => {
  it('keeps a name and a PNG picture', () => {
    const avatar = 'data:image/png;base64,AAAA'
    expect(cleanProfile({ name: '  Dang  ', avatar })).toEqual({ name: 'Dang', avatar })
  })

  it('drops junk', () => {
    expect(cleanProfile(null)).toEqual({ name: '', avatar: '' })
    expect(cleanProfile({ name: 3, avatar: 'https://x/y.png' })).toEqual({ name: '', avatar: '' })
    expect(cleanProfile({ name: 'x'.repeat(100) }).name).toHaveLength(MAX_NAME)
  })
})
