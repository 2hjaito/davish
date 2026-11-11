import { describe, expect, it } from 'vitest'

import { apps, availableOn, getApp, isAvailable } from './registry'

describe('isAvailable', () => {
  it('lets every app try on a server whose OS is not known', () => {
    for (const a of apps) {
      expect(isAvailable(a, 'unknown')).toBe(true)
      expect(isAvailable(a, undefined)).toBe(true)
    }
  })

  it('keeps every app on Linux servers', () => {
    for (const a of apps) expect(isAvailable(a, 'linux'), a.id).toBe(true)
  })

  it('keeps every app on macOS servers, as before Windows support', () => {
    for (const a of apps) expect(isAvailable(a, 'darwin'), a.id).toBe(true)
  })

  it('keeps POSIX-only apps off Windows servers', () => {
    for (const id of [
      'docker',
      'git',
      'keys',
      'ports',
      'deploy',
      'database',
      'services',
      'nginx',
      'cron',
      'logs',
      'packages',
      'disk',
      'security',
    ] as const) {
      expect(isAvailable(getApp(id), 'windows'), id).toBe(false)
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

  it('offers Monitor on every OS, so a macOS server gets an answer instead of a hang', () => {
    const monitor = getApp('monitor')
    for (const os of ['linux', 'windows', 'darwin'] as const) {
      expect(isAvailable(monitor, os), os).toBe(true)
    }
  })

  it('filters a list for the Dock', () => {
    const windows = availableOn(apps, 'windows').map((a) => a.id)
    expect(windows).toContain('terminal')
    expect(windows).not.toContain('docker')
    expect(availableOn(apps, 'darwin')).toHaveLength(apps.length)
  })
})
