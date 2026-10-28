import { describe, expect, it } from 'vitest'

import { displayAddress, frameURL, parseAddress } from './address'

describe('parseAddress', () => {
  it('reads the forms people type', () => {
    expect(parseAddress('localhost:3000/admin?x=1')).toEqual({ port: 3000, path: '/admin?x=1' })
    expect(parseAddress(':8080')).toEqual({ port: 8080, path: '/' })
    expect(parseAddress('3000/api')).toEqual({ port: 3000, path: '/api' })
    expect(parseAddress('http://127.0.0.1:5173/#/home')).toEqual({ port: 5173, path: '/#/home' })
    expect(parseAddress('localhost')).toEqual({ port: 80, path: '/' })
  })

  it('refuses what is not the server', () => {
    expect(parseAddress('github.com')).toBeUndefined()
    expect(parseAddress('https://localhost:3000')).toBeUndefined()
    expect(parseAddress('localhost:99999')).toBeUndefined()
    expect(parseAddress('')).toBeUndefined()
  })
})

describe('frame and display addresses', () => {
  it('adds the token for the frame and hides it in the bar', () => {
    const url = frameURL(51000, '/admin?x=1#top', 'tok')
    expect(url).toBe('http://127.0.0.1:51000/admin?x=1&__davish=tok#top')
    expect(displayAddress(url, 3000)).toBe('localhost:3000/admin?x=1#top')
    expect(displayAddress('http://127.0.0.1:51000/', 80)).toBe('localhost/')
  })
})
