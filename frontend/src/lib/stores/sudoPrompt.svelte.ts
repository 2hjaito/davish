// The one sudo password prompt (ADR 0018): an app that gets
// SUDO_PASSWORD_REQUIRED asks here, then runs its action again.

export interface SudoRequest {
  serverID: string
  /** The command as it runs, with sudo, shown in the prompt. */
  command: string
  /** The same for a terminal, when the user would rather type it there. */
  terminal: string
  done: (ok: boolean) => void
}

class SudoPrompt {
  request = $state.raw<SudoRequest | null>(null)

  /** Resolves true once the password is kept, false when the user cancels. */
  ask(serverID: string, command: string, terminal = command): Promise<boolean> {
    this.request?.done(false)
    return new Promise((resolve) => {
      this.request = {
        serverID,
        command,
        terminal,
        done: (ok) => {
          this.request = null
          resolve(ok)
        },
      }
    })
  }
}

export const sudoPrompt = new SudoPrompt()
