import { expect, it } from 'vitest'

import type { AuditEntry } from '../api/events'
import { CAPACITY, merge } from './audit.svelte'

const e = (id: number): AuditEntry => ({
  id,
  time: id,
  serverID: 's',
  app: 'services',
  command: `cmd ${id}`,
  exit: 0,
  durMs: 1,
})

it('merges in order without duplicates', () => {
  // An event can arrive before the initial load answers.
  expect(merge([e(5)], [e(3), e(4), e(5)]).map((x) => x.id)).toEqual([3, 4, 5])
})

it('keeps the newest entries', () => {
  const many = Array.from({ length: CAPACITY + 10 }, (_, i) => e(i + 1))
  const got = merge([], many)
  expect(got).toHaveLength(CAPACITY)
  expect(got[0]!.id).toBe(11)
})
