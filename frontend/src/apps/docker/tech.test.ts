import { expect, it } from 'vitest'

import type { Container } from '../../lib/api/docker'
import { groups } from './containers'
import { groupTech, imageTech, repository } from './tech'

const c = (name: string, image: string, project?: string): Container => ({
  id: name,
  name,
  image,
  state: 'running',
  status: '',
  ports: '',
  command: '',
  created: '',
  runningFor: '',
  project,
})

it('strips registry, tag and digest', () => {
  expect(
    [
      'postgres:16',
      'library/redis:7-alpine',
      'ghcr.io/foo/bar:1.2',
      'localhost:5000/app',
      'registry.example.com:5000/team/api:dev',
      'mongo@sha256:abc',
    ].map(repository),
  ).toEqual(['postgres', 'redis', 'foo/bar', 'app', 'team/api', 'mongo'])
})

it('knows images by name', () => {
  expect(
    [
      'postgres:16',
      'postgis/postgis:16-3.4',
      'bitnami/redis:7',
      'jc21/nginx-proxy-manager:latest',
      'n8nio/n8n',
      'louislam/uptime-kuma:1',
      'vaultwarden/server:latest',
      'node:22-alpine',
      'mcr.microsoft.com/dotnet/aspnet:8.0',
      'myapp/api:2.4.1',
      'davitek/zalo-gateway:latest',
      'nodejs-app',
    ].map(imageTech),
  ).toEqual([
    'postgres',
    'postgres',
    'redis',
    'nginx',
    'n8n',
    'uptimekuma',
    'vaultwarden',
    'node',
    'dotnet',
    undefined,
    undefined,
    undefined,
  ])
})

it('gives a stack a logo only when every image is known', () => {
  // groups() sorts projects by name: n8n, shop, then the lone pg.
  const [n8n, shop, db] = groups([
    c('api', 'myapp/api:2', 'shop'),
    c('db', 'postgres:16', 'shop'),
    c('n8n', 'n8nio/n8n', 'n8n'),
    c('n8n-db', 'postgres:16', 'n8n'),
    c('pg', 'postgres:16'),
  ])
  expect(groupTech(n8n!)).toBe('n8n') // the product, not its database
  expect(groupTech(shop!)).toBeUndefined() // the user's own app is unknown
  expect(groupTech(db!)).toBe('postgres')
})
