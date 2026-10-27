import { describe, expect, it } from 'vitest'

import { backupByDefault } from './save'

describe('backupByDefault', () => {
  it('keeps backups of system configuration only', () => {
    expect(backupByDefault('/etc/nginx/nginx.conf')).toBe(true)
    expect(backupByDefault('/home/u/.bashrc')).toBe(false)
    expect(backupByDefault('/etcetera/x')).toBe(false)
  })
})
