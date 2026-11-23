import { describe, expect, it } from 'vitest'

import { WIDGET_KINDS } from './kinds'
import { widgetAvailable, widgetDefs } from './registry'

describe('widgetDefs', () => {
  it('has an entry, a label and at least one size for every kind', () => {
    for (const kind of WIDGET_KINDS) {
      const def = widgetDefs[kind]
      expect(def.label()).not.toBe('')
      expect(def.sizes.length).toBeGreaterThan(0)
    }
  })
})

describe('widgetAvailable', () => {
  it('follows the app a widget opens: Docker is not on a Windows server', () => {
    expect(widgetAvailable('docker', 'linux')).toBe(true)
    expect(widgetAvailable('docker', 'darwin')).toBe(true)
    expect(widgetAvailable('docker', 'windows')).toBe(false)
  })

  it('has Monitor everywhere, and everything for a server of unknown system', () => {
    for (const os of ['linux', 'darwin', 'windows'] as const)
      expect(widgetAvailable('monitor', os)).toBe(true)
    for (const kind of WIDGET_KINDS) {
      expect(widgetAvailable(kind, undefined)).toBe(true)
      expect(widgetAvailable(kind, 'unknown')).toBe(true)
    }
  })
})
