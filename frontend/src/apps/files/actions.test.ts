import { describe, expect, it } from 'vitest'

import { actionForKey, petalsFor, renameProblem, stemLength } from './actions'

const key = (over: Partial<Parameters<typeof actionForKey>[0]>) => ({
  key: '',
  code: '',
  primary: false,
  shift: false,
  alt: false,
  ...over,
})

describe('petalsFor', () => {
  it('fits files and folders', () => {
    expect(petalsFor({ dir: false })).toEqual(['editor', 'rename', 'delete', 'copyPath'])
    expect(petalsFor({ dir: true })).toEqual(['terminal', 'rename', 'delete', 'compress'])
  })
})

describe('actionForKey', () => {
  it('maps the shortcuts', () => {
    expect(actionForKey(key({ key: 'F2' }))).toBe('rename')
    expect(actionForKey(key({ key: 'Delete' }))).toBe('delete')
    expect(actionForKey(key({ key: 'Backspace', primary: true }))).toBe('delete')
    expect(actionForKey(key({ key: 't', primary: true }))).toBe('terminal')
    // Option+C types "ç" on a Mac; the physical key still counts.
    expect(actionForKey(key({ key: 'ç', code: 'KeyC', primary: true, alt: true }))).toBe('copyPath')
    expect(actionForKey(key({ key: 'Ω', code: 'KeyZ', primary: true, alt: true }))).toBe('compress')
  })

  it('ignores everything else', () => {
    expect(actionForKey(key({ key: 'Backspace' }))).toBeNull()
    expect(actionForKey(key({ key: 't' }))).toBeNull()
    expect(actionForKey(key({ key: 'T', primary: true, shift: true }))).toBeNull()
    expect(actionForKey(key({ key: 'F2', primary: true }))).toBeNull()
  })
})

describe('renameProblem', () => {
  it('accepts a new plain name', () => {
    expect(renameProblem('a.txt', ' b.txt ')).toBeNull()
  })

  it('names what is wrong', () => {
    expect(renameProblem('a', '  ')).toBe('empty')
    expect(renameProblem('a', '..')).toBe('empty')
    expect(renameProblem('a', 'x/y')).toBe('slash')
    expect(renameProblem('a', 'a')).toBe('same')
  })
})

describe('stemLength', () => {
  it('stops before the extension, keeping dotfiles whole', () => {
    expect(stemLength('nginx.conf')).toBe(5)
    expect(stemLength('.bashrc')).toBe(7)
    expect(stemLength('Makefile')).toBe(8)
  })
})
