import { expect, it } from 'vitest'

import { pushRecent } from './recentApps.svelte'

it('moves an app to the front, once, keeping five', () => {
  expect(pushRecent([], 'git')).toEqual(['git'])
  expect(pushRecent(['git', 'docker'], 'docker')).toEqual(['docker', 'git'])
  expect(pushRecent(['a', 'b', 'c', 'd', 'e'], 'f')).toEqual(['f', 'a', 'b', 'c', 'd'])
  expect(pushRecent(['a', 'b', 'c', 'd', 'e'], 'd')).toEqual(['d', 'a', 'b', 'c', 'e'])
})
