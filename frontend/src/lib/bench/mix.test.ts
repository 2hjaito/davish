import { describe, expect, it } from 'vitest'

import { widgetMix } from './mix'

describe('widgetMix', () => {
  it('is empty for none, and for what is not a count', () => {
    expect(widgetMix(0)).toEqual([])
    expect(widgetMix(-2)).toEqual([])
    expect(widgetMix(Number.NaN)).toEqual([])
  })

  it('starts with the heaviest widget', () => {
    expect(widgetMix(1)).toEqual([['monitor', 'M']])
    expect(widgetMix(3)).toEqual([
      ['monitor', 'M'],
      ['docker', 'M'],
      ['monitor', 'S'],
    ])
  })

  it('repeats the mix past what it holds', () => {
    const six = widgetMix(6)
    expect(six).toHaveLength(6)
    expect(six.slice(3)).toEqual(widgetMix(3))
  })
})
