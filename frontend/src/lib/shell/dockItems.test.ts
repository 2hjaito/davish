import { describe, expect, it } from 'vitest'

import { dockApps } from '../../apps/registry'
import { DEFAULT_PINS, dockItems, moveItem, movePin, togglePin } from './dockItems'

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

describe('movePin', () => {
  it('moves a pin to just before another', () => {
    expect(movePin(['a', 'b', 'c', 'd'], 'd', 'b')).toEqual(['a', 'd', 'b', 'c'])
    expect(movePin(['a', 'b', 'c', 'd'], 'a', 'c')).toEqual(['b', 'a', 'c', 'd'])
  })

  it('moves a pin to the end when nothing follows it', () => {
    expect(movePin(['a', 'b', 'c'], 'a', null)).toEqual(['b', 'c', 'a'])
    expect(movePin(['a', 'b', 'c'], 'a', 'running')).toEqual(['b', 'c', 'a'])
  })

  it('keeps pins the dock is not showing in their places', () => {
    // 'win' is pinned but hidden on this server; 'c' is dropped before 'b'.
    expect(movePin(['a', 'win', 'b', 'c'], 'c', 'b')).toEqual(['a', 'win', 'c', 'b'])
  })

  it('pins an app that was not pinned', () => {
    expect(movePin(['a', 'b'], 'x', 'b')).toEqual(['a', 'x', 'b'])
  })

  it('leaves the pins as they were when dropped in place', () => {
    expect(movePin(['a', 'b', 'c'], 'b', 'c')).toEqual(['a', 'b', 'c'])
  })
})

describe('moveItem', () => {
  it('moves an item forward and back', () => {
    expect(moveItem(['a', 'b', 'c', 'd'], 1, 3)).toEqual(['a', 'c', 'd', 'b'])
    expect(moveItem(['a', 'b', 'c', 'd'], 3, 1)).toEqual(['a', 'd', 'b', 'c'])
    expect(moveItem(['a', 'b'], 1, 1)).toEqual(['a', 'b'])
  })
})

describe('DEFAULT_PINS', () => {
  it('names only apps the dock can show', () => {
    const ids = dockApps.map((a) => a.id)
    for (const id of DEFAULT_PINS) expect(ids).toContain(id)
  })
})
