import { describe, expect, it } from 'vitest'

import { backupFor } from './save'

describe('backupFor', () => {
  it('keeps backups of system configuration by default', () => {
    expect(backupFor('/etc/nginx/nginx.conf')).toBe(true)
    expect(backupFor('/home/u/.bashrc')).toBe(false)
    expect(backupFor('/etcetera/x')).toBe(false)
  })

  it('follows Settings', () => {
    expect(backupFor('/home/u/.bashrc', 'always')).toBe(true)
    expect(backupFor('/etc/hosts', 'never')).toBe(false)
  })
})
