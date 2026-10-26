import { beforeEach, describe, expect, it, vi } from 'vitest'

import { MockShell } from './mockShell'

// Strips ANSI so assertions read like what the user sees.
// eslint-disable-next-line no-control-regex -- matching ESC is the point
const plain = (s: string) => s.replace(/\x1b\[[0-9;]*[A-Za-z]/g, '')

describe('MockShell', () => {
  let output: string
  let shell: MockShell
  const onExit = vi.fn()
  const onCwd = vi.fn()

  beforeEach(() => {
    output = ''
    onExit.mockReset()
    onCwd.mockReset()
    shell = new MockShell({
      user: 'dangth',
      host: 'tsc-vps',
      banner: 'hi',
      emit: (d) => (output += d),
      onExit,
      onCwd,
    })
    shell.start()
    output = ''
  })

  const type = (s: string) => shell.input(s + '\r')

  it('echoes typed characters and runs commands', () => {
    type('echo xin chào')
    expect(plain(output)).toBe('echo xin chào\r\nxin chào\r\ndangth@tsc-vps:~$ ')
  })

  it('changes directory and reports the new cwd', () => {
    type('cd /etc/nginx')
    expect(shell.cwd).toBe('/etc/nginx')
    expect(onCwd).toHaveBeenCalledWith('/etc/nginx')
    type('cd ..')
    expect(shell.cwd).toBe('/etc')
  })

  it('rejects unknown directories and commands', () => {
    type('cd /nope')
    type('rm -rf /')
    expect(plain(output)).toContain('No such file or directory')
    expect(plain(output)).toContain('bash: rm: command not found')
    expect(shell.cwd).toBe('/home/dangth')
  })

  it('handles backspace before running', () => {
    shell.input('pwdd')
    shell.input('\x7f')
    shell.input('\r')
    expect(plain(output)).toContain('\r\n/home/dangth\r\n')
  })

  it('recalls history with the up arrow', () => {
    type('whoami')
    output = ''
    shell.input('\x1b[A')
    expect(plain(output)).toBe('whoami')
  })

  it('cancels the line on Ctrl+C', () => {
    shell.input('ls')
    shell.input('\x03')
    expect(plain(output)).toBe('ls^C\r\ndangth@tsc-vps:~$ ')
  })

  it('exits on exit', () => {
    type('exit')
    expect(onExit).toHaveBeenCalled()
  })
})
