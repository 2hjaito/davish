import { describe, expect, it } from 'vitest'

import { absoluteCwd, cwdFromTitle, parseOsc7 } from './cwd'

describe('parseOsc7', () => {
  it('reads the path after the host', () => {
    expect(parseOsc7('file://vps/var/www/shop')).toBe('/var/www/shop')
    expect(parseOsc7('file:///srv')).toBe('/srv')
  })

  it('decodes percent escapes', () => {
    expect(parseOsc7('file://vps/srv/100%25%20done')).toBe('/srv/100% done')
  })

  it('keeps a path that is not valid escaping', () => {
    expect(parseOsc7('file://vps/srv/100%')).toBe('/srv/100%')
  })

  it('drops anything else', () => {
    expect(parseOsc7('http://vps/srv')).toBeNull()
    expect(parseOsc7('file://vps')).toBeNull()
    expect(parseOsc7('')).toBeNull()
  })
})

describe('cwdFromTitle', () => {
  it('reads the folder of a shell title', () => {
    expect(cwdFromTitle('dang@vps: ~/projects')).toBe('~/projects')
    expect(cwdFromTitle('root@vps:/etc/nginx')).toBe('/etc/nginx')
    expect(cwdFromTitle('dang@vps: /srv/my app')).toBe('/srv/my app')
  })

  it('ignores the titles of programs', () => {
    expect(cwdFromTitle('vim notes.txt')).toBeNull()
    expect(cwdFromTitle('htop')).toBeNull()
    expect(cwdFromTitle('dang@vps: running tests')).toBeNull()
  })
})

describe('absoluteCwd', () => {
  it('expands home', () => {
    expect(absoluteCwd('~', '/home/d')).toBe('/home/d')
    expect(absoluteCwd('~/a', '/home/d')).toBe('/home/d/a')
    expect(absoluteCwd('/srv', '/home/d')).toBe('/srv')
  })

  it('refuses what is not a path', () => {
    expect(absoluteCwd('~other/a', '/home/d')).toBeNull()
    expect(absoluteCwd('a/b', '/home/d')).toBeNull()
  })
})
