import { describe, expect, it } from 'vitest'

import { findPaths } from './logPaths'

const paths = (line: string) =>
  findPaths(line)
    .filter((s) => s.path)
    .map((s) => [s.path, s.line])

describe('findPaths', () => {
  it.each([
    [
      'Error: cannot open /etc/nginx/nginx.conf: no such file',
      [['/etc/nginx/nginx.conf', undefined]],
    ],
    [
      'nginx: [emerg] unknown directive "x" in /etc/nginx/sites-enabled/default:12',
      [['/etc/nginx/sites-enabled/default', 12]],
    ],
    ['  File "/app/worker.py", line 42, in run', [['/app/worker.py', 42]]],
    ['panic at /src/main.go:88:5 (goroutine 1)', [['/src/main.go', 88]]],
    [
      'wrote /var/log/a.log and /var/log/b.log.',
      [
        ['/var/log/a.log', undefined],
        ['/var/log/b.log', undefined],
      ],
    ],
    ['GET https://example.com/api/v1/users 200', []],
    ['moved ~/notes/today.md', []],
    ['ratio 3/4 done', []],
    ['the root / is full', []],
    ['/usr/bin/env', [['/usr/bin/env', undefined]]],
    ['GET /api/orders 200 12ms', []],
    ['GET /static/js/app.js 304', [['/static/js/app.js', undefined]]],
    ['at /build/src/index.ts:10', [['/build/src/index.ts', 10]]],
  ])('%s', (line, want) => expect(paths(line)).toEqual(want))

  it('keeps the whole line in order', () => {
    const line = 'x /etc/hosts:3 y'
    expect(
      findPaths(line)
        .map((s) => s.text)
        .join(''),
    ).toBe(line)
    expect(findPaths(line)).toEqual([
      { text: 'x ' },
      { text: '/etc/hosts:3', path: '/etc/hosts', line: 3 },
      { text: ' y' },
    ])
  })
})
