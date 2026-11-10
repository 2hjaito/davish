import { describe, expect, it } from 'vitest'

import { sudoPrompt } from './sudoPrompt.svelte'

describe('sudoPrompt', () => {
  it('answers every action waiting on the same server with one password', async () => {
    const a = sudoPrompt.ask('srv', 'sudo systemctl restart nginx')
    const b = sudoPrompt.ask('srv', 'sudo systemctl stop redis')
    expect(sudoPrompt.request?.command).toBe('sudo systemctl restart nginx')
    sudoPrompt.request?.done(true)
    expect(await a).toBe(true)
    expect(await b).toBe(true)
    expect(sudoPrompt.request).toBeNull()
  })

  it('asks for the next server once the first is answered', async () => {
    const a = sudoPrompt.ask('one', 'sudo true')
    const b = sudoPrompt.ask('two', 'sudo false')
    expect(sudoPrompt.request?.serverID).toBe('one')
    sudoPrompt.request?.done(false)
    expect(await a).toBe(false)
    expect(sudoPrompt.request?.serverID).toBe('two')
    sudoPrompt.request?.done(true)
    expect(await b).toBe(true)
  })
})
