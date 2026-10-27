import { describe, expect, it } from 'vitest'

import { cameFrom } from './memory'

describe('cameFrom', () => {
  it('names the child leading back to where we were', () => {
    expect(cameFrom('/etc/nginx/sites-enabled', '/etc')).toBe('/etc/nginx')
    expect(cameFrom('/etc/nginx', '/etc')).toBe('/etc/nginx')
    expect(cameFrom('/etc', '/')).toBe('/etc')
  })

  it('is null when not going up', () => {
    expect(cameFrom('/etc', '/etc/nginx')).toBeNull()
    expect(cameFrom('/etc', '/etc')).toBeNull()
    expect(cameFrom('/etcetera', '/etc')).toBeNull()
    expect(cameFrom('/var/log', '/etc')).toBeNull()
  })
})
