import { expect, it } from 'vitest'

import { mountOf, useByMount } from './dockerDisks'

const mounts = ['/', '/boot/efi', '/mnt/data', '/mnt/data2']

it('finds the longest mount point', () => {
  expect(mountOf('/var/lib/docker', mounts)).toBe('/')
  expect(mountOf('/mnt/data/shop', mounts)).toBe('/mnt/data')
  expect(mountOf('/mnt/data2/x', mounts)).toBe('/mnt/data2')
  expect(mountOf('/mnt/data', mounts)).toBe('/mnt/data')
  expect(mountOf('/mnt/database', mounts)).toBe('/') // not a prefix by folder
  expect(mountOf('/x', ['/mnt'])).toBeUndefined()
})

it('groups owners by mount', () => {
  const use = useByMount(
    {
      version: '27',
      rootDir: '/var/lib/docker',
      mounts: [
        {
          container: 'shop-db-1',
          project: 'shop',
          type: 'volume',
          source: '/var/lib/docker/volumes/a',
          dest: '/d',
        },
        {
          container: 'shop-redis-1',
          project: 'shop',
          type: 'volume',
          source: '/var/lib/docker/volumes/b',
          dest: '/d',
        },
        { container: 'gateway', type: 'bind', source: '/mnt/data/zalo', dest: '/d' },
        {
          container: 'shop-api-1',
          project: 'shop',
          type: 'bind',
          source: '/mnt/data/shop',
          dest: '/d',
        },
      ],
    },
    mounts,
  )
  expect(use.get('/')).toEqual({ root: true, owners: ['shop'] })
  expect(use.get('/mnt/data')).toEqual({ root: false, owners: ['gateway', 'shop'] })
  expect(use.get('/boot/efi')).toBeUndefined()
})
