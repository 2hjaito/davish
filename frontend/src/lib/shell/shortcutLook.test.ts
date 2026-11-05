import { describe, expect, it } from 'vitest'

import { iconImage, iconOf, labelFor, squareCrop, titleOf } from './shortcutLook'

describe('titleOf', () => {
  it('prefers the label and falls back to the file name', () => {
    expect(titleOf({ name: 'deploy.sh' })).toBe('deploy.sh')
    expect(titleOf({ name: 'deploy.sh', label: 'Ship it' })).toBe('Ship it')
  })
})

describe('iconOf', () => {
  it('follows the kind until the user picks one', () => {
    expect(iconOf({ kind: 'script' })).toBe('terminal')
    expect(iconOf({ kind: 'folder' })).toBe('folder')
    expect(iconOf({ kind: 'script', icon: 'zap' })).toBe('zap')
  })
})

describe('labelFor', () => {
  it('keeps a real label, trimmed', () => {
    expect(labelFor('  Ship it ', 'deploy.sh')).toBe('Ship it')
  })

  it('treats blank and the file name as no label', () => {
    expect(labelFor('   ', 'deploy.sh')).toBeUndefined()
    expect(labelFor(' deploy.sh ', 'deploy.sh')).toBeUndefined()
  })
})

describe('squareCrop', () => {
  it('centers the square on the long side', () => {
    expect(squareCrop(400, 200)).toEqual({ x: 100, y: 0, side: 200 })
    expect(squareCrop(200, 400)).toEqual({ x: 0, y: 100, side: 200 })
    expect(squareCrop(64, 64)).toEqual({ x: 0, y: 0, side: 64 })
  })
})

describe('iconImage', () => {
  it('refuses SVG, which can carry scripts', async () => {
    const svg = new File(['<svg/>'], 'logo.svg', { type: 'image/svg+xml' })
    await expect(iconImage(svg)).rejects.toThrow('unsupported image')
  })

  it('refuses a file that is not an image', async () => {
    const text = new File(['hello'], 'notes.txt', { type: 'text/plain' })
    await expect(iconImage(text)).rejects.toThrow('unsupported image')
  })

  it('refuses a huge picture', async () => {
    const huge = new File([new Uint8Array(21 << 20)], 'big.png', { type: 'image/png' })
    await expect(iconImage(huge)).rejects.toThrow('unsupported image')
  })
})
