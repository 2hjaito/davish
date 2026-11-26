import { describe, expect, it } from 'vitest'

import { osIcons } from './osIcons'

// The keys internal/platform/distro.go can give a server (logos, plus the
// plain Linux, Windows and macOS ones the connection adds). A key without a
// logo would leave a blank box in the lists.
const keys = [
  'ubuntu',
  'debian',
  'fedora',
  'centos',
  'rocky',
  'almalinux',
  'rhel',
  'arch',
  'alpine',
  'opensuse',
  'raspberrypi',
  'kali',
  'linuxmint',
  'linux',
  'windows',
  'macos',
]

describe('osIcons', () => {
  it.each(keys)('has a logo for %s', (key) => {
    const logo = (osIcons as Record<string, { viewBox: string; nodes: unknown[] } | undefined>)[key]
    expect(logo?.nodes.length).toBeGreaterThan(0)
    expect(logo?.viewBox).toBe('0 0 24 24')
  })

  it('has nothing beyond those keys', () => {
    expect(Object.keys(osIcons).sort()).toEqual([...keys].sort())
  })
})
