import { describe, expect, it } from 'vitest'

import { icons } from './icons'

describe('icons', () => {
  it.each(Object.entries(icons))('%s has a viewBox and nodes', (_name, def) => {
    expect(def.viewBox).toMatch(/^[\d.\s-]+$/)
    expect(def.nodes.length).toBeGreaterThan(0)
  })
})
