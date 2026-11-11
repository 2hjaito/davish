import { describe, expect, it } from 'vitest'

import { sudoPrompt, withSudo } from './sudoPrompt.svelte'

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

  it('offers Terminal a different line when the command reads stdin', async () => {
    let calls = 0
    const act = () => {
      calls++
      if (calls === 1) return Promise.reject({ code: 'SUDO_PASSWORD_REQUIRED' })
      return Promise.resolve()
    }
    const run = withSudo('srv', "sudo sh -c 'crontab -u root -'", act, 'sudo crontab -u root -e')
    await Promise.resolve()
    await Promise.resolve()
    expect(sudoPrompt.request?.command).toBe("sudo sh -c 'crontab -u root -'")
    expect(sudoPrompt.request?.terminal).toBe('sudo crontab -u root -e')
    sudoPrompt.request?.done(true)
    expect(await run).toBe(true)
    expect(calls).toBe(2)
  })
})
