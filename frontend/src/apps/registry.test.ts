import { describe, expect, it } from 'vitest'

import { apps, availableOn, getApp, isAvailable } from './registry'

describe('isAvailable', () => {
  it('lets every app try on a server whose OS is not known', () => {
    for (const a of apps) {
      expect(isAvailable(a, 'unknown')).toBe(true)
      expect(isAvailable(a, undefined)).toBe(true)
    }
  })

  it('keeps Linux-only apps off Windows and macOS servers', () => {
    for (const id of ['docker', 'services', 'nginx', 'cron', 'logs'] as const) {
      const app = getApp(id)
      expect(isAvailable(app, 'linux')).toBe(true)
      expect(isAvailable(app, 'windows')).toBe(false)
      expect(isAvailable(app, 'darwin')).toBe(false)
    }
  })

  it('keeps the apps that only need SSH and SFTP on every OS', () => {
    for (const id of ['terminal', 'files', 'editor', 'transfers', 'settings'] as const) {
      const app = getApp(id)
      for (const os of ['linux', 'windows', 'darwin'] as const) {
        expect(isAvailable(app, os), `${id} on ${os}`).toBe(true)
      }
    }
  })

  it('offers Monitor on Linux and Windows only', () => {
    const monitor = getApp('monitor')
    expect(isAvailable(monitor, 'linux')).toBe(true)
    expect(isAvailable(monitor, 'windows')).toBe(true)
    expect(isAvailable(monitor, 'darwin')).toBe(false)
  })

  it('filters a list for the Dock', () => {
    const ids = availableOn(apps, 'windows').map((a) => a.id)
    expect(ids).toContain('terminal')
    expect(ids).not.toContain('docker')
  })
})
