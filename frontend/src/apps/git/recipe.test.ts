import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { DeployInfo } from '../../lib/api/git'

const store = vi.hoisted(() => new Map<string, unknown>())
vi.mock('../../lib/stores/persist', () => ({
  load: <T>(k: string, fallback: T): T => (store.get(k) as T | undefined) ?? fallback,
  save: (k: string, v: unknown) => store.set(k, structuredClone(v)),
}))

const { loadPlan, savePlan, stepsOf, suggestPlan } = await import('./recipe')
type StepLabels = import('./recipe').StepLabels

const opt = (name: string, command: string, sudo = false) => ({ name, command, sudo })

const node: DeployInfo = {
  pull: opt('git pull', 'git pull --ff-only'),
  install: opt('npm', 'npm ci'),
  builds: [opt('build', 'npm run build'), opt('lint', 'npm run lint')],
  runners: [
    { kind: 'pm2', ...opt('shop', 'pm2 restart shop') },
    { kind: 'systemd', ...opt('shop.service', 'sudo systemctl restart shop.service', true) },
  ],
  compose: false,
}

const compose: DeployInfo = {
  ...node,
  runners: [{ kind: 'compose', ...opt('shop', 'docker compose up -d --build') }],
  compose: true,
}

const labels: StepLabels = {
  pull: 'Pull',
  install: 'Install',
  build: (n) => `Build ${n}`,
  restart: (k, n) => `Restart ${k} ${n}`,
}

describe('deploy plans', () => {
  beforeEach(() => store.clear())

  it('suggests every step the repository points to', () => {
    expect(suggestPlan(node)).toEqual({
      pull: true,
      install: true,
      build: 'build',
      restart: 'pm2:shop',
      custom: [],
    })
    expect(stepsOf(suggestPlan(node), node, labels)).toEqual([
      { label: 'Pull', command: 'git pull --ff-only' },
      { label: 'Install', command: 'npm ci' },
      { label: 'Build build', command: 'npm run build' },
      { label: 'Restart pm2 shop', command: 'pm2 restart shop' },
    ])
  })

  it('leaves install and build to compose', () => {
    expect(suggestPlan(compose)).toMatchObject({
      install: false,
      build: '',
      restart: 'compose:shop',
    })
  })

  it('runs custom steps before the restart', () => {
    const plan = {
      ...suggestPlan(node),
      install: false,
      build: '',
      custom: [' npm run migrate ', ''],
    }
    expect(stepsOf(plan, node, labels).map((s) => s.command)).toEqual([
      'git pull --ff-only',
      'npm run migrate',
      'pm2 restart shop',
    ])
  })

  it('loads a saved plan and drops what the server no longer has', () => {
    expect(loadPlan('s', '/srv/shop', node).saved).toBe(false)
    savePlan('s', '/srv/shop', { ...suggestPlan(node), restart: 'pm2:gone', build: 'lint' })
    const l = loadPlan('s', '/srv/shop', node)
    expect(l).toMatchObject({ saved: true, lostRestart: 'gone' })
    expect(l.plan).toMatchObject({ restart: '', build: 'lint' })
  })

  it('keeps an old command recipe as custom steps', () => {
    store.set('deploy-recipes', { 's:/srv/shop': ['git pull --ff-only', '# hint', 'make deploy'] })
    expect(loadPlan('s', '/srv/shop', node).plan).toEqual({
      pull: false,
      install: false,
      build: '',
      restart: '',
      custom: ['git pull --ff-only', 'make deploy'],
    })
  })
})
