import { describe, expect, it } from 'vitest'

import {
  IDLE_MS,
  finished,
  keepTail,
  lastLine,
  mockSteps,
  nextActive,
  promptShown,
  sudoRefused,
  tally,
  waiting,
} from './run'

describe('multi-server run', () => {
  it('reads the last line as shown', () => {
    expect(lastLine('a\nb\x1b[1mold\rnew')).toBe('new')
    expect(lastLine('done\n')).toBe('')
    expect(lastLine('\x1b]0;title\x07[sudo] password for me: ')).toBe('[sudo] password for me: ')
  })

  it('spots prompts', () => {
    for (const p of [
      '[sudo] password for deploy: ',
      'Password: ',
      'Do you want to continue? [Y/n] ',
      'Is this ok [y/N]: ',
      'Are you sure you want to continue connecting (yes/no/[fingerprint])? ',
      'Enter passphrase for key /x: ',
    ])
      expect(promptShown(`line\n${p}`), p).toBe(true)
    expect(promptShown('Reading package lists... 50%')).toBe(false)
    expect(promptShown('password changed\n')).toBe(false)
  })

  it('counts a quiet, unfinished line as waiting, only while running', () => {
    expect(waiting('running', 'Continue', IDLE_MS)).toBe(true)
    expect(waiting('running', 'Continue', 200)).toBe(false)
    expect(waiting('running', 'done\n', 10_000)).toBe(false)
    expect(waiting('running', '[sudo] password for a: ', 0)).toBe(true)
    expect(waiting('done', '[sudo] password for a: ', 0)).toBe(false)
  })

  it('keeps only the end of the output', () => {
    expect(keepTail('x'.repeat(500), 'end').endsWith('end')).toBe(true)
    expect(keepTail('x'.repeat(500), 'end').length).toBe(400)
  })

  it('tells when sudo -n refused', () => {
    expect(sudoRefused('sudo: a password is required\r\n')).toBe(true)
    expect(sudoRefused('[sudo] password for a:')).toBe(false)
  })

  it('tallies and knows when it is over', () => {
    const jobs = [{ state: 'done' }, { state: 'failed' }, { state: 'running' }] as const
    expect(tally(jobs)).toEqual({ queued: 0, running: 1, done: 1, failed: 1, stopped: 0 })
    expect(finished(jobs)).toBe(false)
    expect(finished([{ state: 'done' }, { state: 'stopped' }])).toBe(true)
  })

  it('brings the first waiting tab forward unless the shown one waits', () => {
    const w = new Set(['b', 'c'])
    expect(nextActive(['a', 'b', 'c'], (id) => w.has(id), 'a')).toBe('b')
    expect(nextActive(['a', 'b', 'c'], (id) => w.has(id), 'c')).toBe('c')
    expect(nextActive(['a', 'b'], () => false, 'a')).toBe('a')
  })

  it('scripts a preview run with one sudo prompt', () => {
    const steps = mockSteps(['a', 'b', 'c'], 'sudo apt-get update')
    expect(steps.some((s) => s.serverID === 'b' && s.out?.includes('[sudo]'))).toBe(true)
    expect(steps.filter((s) => s.state === 'done').length).toBe(1)
  })
})
