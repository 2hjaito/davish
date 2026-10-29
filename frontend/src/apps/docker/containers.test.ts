import { describe, expect, it } from 'vitest'

import type { Container } from '../../lib/api/docker'
import {
  ago,
  canDo,
  clock,
  counts,
  groups,
  health,
  parsePorts,
  since,
  visible,
  visibleImages,
} from './containers'

const c = (name: string, state: string, image = 'nginx:1.27'): Container => ({
  id: name,
  name,
  image,
  state,
  status: '',
  ports: '',
  command: '',
  created: '',
  runningFor: '',
})

const list = [
  c('web', 'running'),
  c('worker', 'restarting', 'myapp:2'),
  c('old', 'exited', 'postgres:16'),
  c('new', 'created'),
  c('held', 'paused'),
]

it('colors states', () => {
  expect(['running', 'paused', 'restarting', 'dead', 'exited', 'created'].map(health)).toEqual([
    'ok',
    'busy',
    'failed',
    'failed',
    'off',
    'off',
  ])
})

describe('filters', () => {
  it('counts each filter', () => {
    expect(counts(list)).toEqual({ all: 5, running: 2, restarting: 1, stopped: 2 })
  })

  it('searches name and image', () => {
    expect(visible(list, 'all', 'POSTGRES').map((x) => x.name)).toEqual(['old'])
    expect(visible(list, 'running', 'nginx').map((x) => x.name)).toEqual(['web', 'held'])
    expect(visible(list, 'restarting', '')).toHaveLength(1)
  })
})

it('enables actions by state', () => {
  expect(canDo('exited', 'start')).toBe(true)
  expect(canDo('running', 'start')).toBe(false)
  expect(canDo('restarting', 'stop')).toBe(true)
  expect(canDo('created', 'stop')).toBe(false)
  expect(canDo('exited', 'restart')).toBe(true)
  expect(canDo('removing', 'restart')).toBe(false)
})

it('measures time since a docker timestamp', () => {
  const now = Date.parse('2026-10-27T16:00:00Z')
  expect(since('2026-10-27T15:00:00.123456789Z', now)).toBe(3599)
  expect(since('', now)).toBe(0)
  expect(since('2026-10-28T00:00:00Z', now)).toBe(0)
})

it('shows log times as a local clock', () => {
  const d = new Date(2026, 9, 27, 8, 5, 9)
  expect(clock(d.toISOString())).toBe('08:05:09')
  expect(clock('garbage')).toBe('garbage')
})

describe('visibleImages', () => {
  const img = (repository: string, containers: number) => ({
    id: `sha256:${repository}`,
    repository,
    tag: 'latest',
    created: '',
    createdSince: '',
    size: 1,
    containers,
    dangling: false,
  })
  const list = [img('nginx', 2), img('old-api', 0)]

  it('keeps unused images for that filter and matches name or ID', () => {
    expect(visibleImages(list, 'unused', '').map((i) => i.repository)).toEqual(['old-api'])
    expect(visibleImages(list, 'images', 'NGINX:lat').map((i) => i.repository)).toEqual(['nginx'])
    expect(visibleImages(list, 'images', 'sha256:old')).toHaveLength(1)
  })
})

describe('ago', () => {
  it('reads docker durations in Vietnamese', () => {
    expect(ago('6 days ago')).toBe('6 ngày trước')
    expect(ago('About an hour ago')).toBe('khoảng 1 giờ trước')
    expect(ago('a minute ago')).toBe('1 phút trước')
    expect(ago('Less than a second ago')).toBe('vừa xong')
    expect(ago('2026-10-28')).toBe('2026-10-28')
  })
})

describe('parsePorts', () => {
  it('keeps one chip per port and tells loopback apart', () => {
    expect(
      parsePorts('0.0.0.0:8080->80/tcp, :::8080->80/tcp, 5432/tcp, 127.0.0.1:3000->3000/tcp'),
    ).toEqual([
      { host: 8080, container: '80', local: false },
      { container: '5432', local: false },
      { host: 3000, container: '3000', local: true },
    ])
    expect(parsePorts('')).toEqual([])
  })
})

describe('groups', () => {
  it('puts compose projects first, then lone containers', () => {
    const c = (name: string, project?: string) => ({ name, project }) as Container
    const g = groups([
      c('migrate'),
      c('shop-api-1', 'shop'),
      c('shop-db-1', 'shop'),
      c('a-1', 'alpha'),
    ])
    expect(g.map((x) => x.project ?? x.key)).toEqual(['alpha', 'shop', 'migrate'])
    expect(g[1]?.containers).toHaveLength(2)
  })
})
