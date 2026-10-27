import { describe, expect, it } from 'vitest'

import { lineLevel } from './logLevel'

describe('lineLevel', () => {
  it.each([
    ['sh[34]: error: cannot open /etc/nginx/nginx.conf', 'error'],
    ["x.service: Failed with result 'exit-code'.", 'error'],
    ['nginx: [emerg] unknown directive', 'error'],
    ['PHP Warning: deprecated', 'warn'],
    ['Started nginx.service', null],
    ['terrorist mode off', null],
  ])('%s', (line, want) => expect(lineLevel(line)).toBe(want))
})
