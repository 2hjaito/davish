import { beforeEach, describe, expect, it, vi } from 'vitest'

import { settings } from '../stores/settings.svelte'
import { widgets } from '../stores/widgets.svelte'
import { reservedFor } from './actions'

const S = 'server-1'
const grid = { cols: 8, rows: 6 }

const memory = new Map<string, string>()
vi.stubGlobal('localStorage', {
  getItem: (key: string) => memory.get(key) ?? null,
  setItem: (key: string, value: string) => memory.set(key, value),
})

beforeEach(() => {
  widgets.of(S).forEach((w) => widgets.remove(S, w.id))
  settings.usePreset('beautiful')
  widgets.add(S, 'monitor', 'M', grid)
})

describe('reservedFor', () => {
  it('is the cells under the widgets, for the icons to keep clear of', () => {
    expect(reservedFor(S, grid)).toHaveLength(4)
  })

  it('is empty when the performance preset hides widgets, so icons use the whole desktop', () => {
    settings.usePreset('light')
    expect(reservedFor(S, grid)).toEqual([])
    settings.usePreset('beautiful')
  })

  it('is empty on a server with no widgets', () => {
    expect(reservedFor('other', grid)).toEqual([])
  })
})
