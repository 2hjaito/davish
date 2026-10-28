import { describe, expect, it } from 'vitest'

import { dockItems, togglePin } from './dockItems'

const all = [{ id: 'terminal' }, { id: 'files' }, { id: 'logs' }, { id: 'audit' }]

describe('dockItems', () => {
  it('keeps the pinned order', () => {
    const got = dockItems(all, ['logs', 'terminal'], () => false)
    expect(got.map((a) => a.id)).toEqual(['logs', 'terminal'])
  })

  it('adds open apps that are not pinned after the pinned ones', () => {
    const got = dockItems(all, ['terminal'], (id) => id === 'audit' || id === 'terminal')
    expect(got.map((a) => a.id)).toEqual(['terminal', 'audit'])
  })

  it('skips pinned ids of apps that no longer exist', () => {
    const got = dockItems(all, ['gone', 'files'], () => false)
    expect(got.map((a) => a.id)).toEqual(['files'])
  })
})

describe('togglePin', () => {
  it('pins at the end and unpins in place', () => {
    expect(togglePin(['terminal'], 'logs')).toEqual(['terminal', 'logs'])
    expect(togglePin(['terminal', 'logs', 'files'], 'logs')).toEqual(['terminal', 'files'])
  })
})
