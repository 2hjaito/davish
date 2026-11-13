import { expect, it } from 'vitest'

import { sliceLines } from './selection'

const lines = ['first line', 'second line', 'third line']

it('copies within one line', () => {
  expect(sliceLines(lines, { row: 1, offset: 0 }, { row: 1, offset: 6 })).toBe('second')
})

it('copies across lines, keeping the breaks', () => {
  expect(sliceLines(lines, { row: 0, offset: 6 }, { row: 2, offset: 5 })).toBe(
    'line\nsecond line\nthird',
  )
})

it('takes the points in either order', () => {
  expect(sliceLines(lines, { row: 2, offset: 5 }, { row: 0, offset: 6 })).toBe(
    'line\nsecond line\nthird',
  )
})

it('ignores rows past the end', () => {
  expect(sliceLines(lines, { row: 2, offset: 0 }, { row: 9, offset: 3 })).toBe('third line')
})
