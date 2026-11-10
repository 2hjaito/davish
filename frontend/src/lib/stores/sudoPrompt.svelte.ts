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

interface Waiting {
  serverID: string
  command: string
  terminal: string
  resolve: (ok: boolean) => void
}

class SudoPrompt {
  request = $state.raw<SudoRequest | null>(null)
  // Asks for a server already prompting wait for its answer; asks for
  // another server wait their turn. None is dropped.
  #waiting: Waiting[] = []

  /** Resolves true once the password is kept, false when the user cancels. */
  ask(serverID: string, command: string, terminal = command): Promise<boolean> {
    return new Promise((resolve) => {
      this.#waiting.push({ serverID, command, terminal, resolve })
      if (!this.request) this.#next()
    })
  }

  #next() {
    const first = this.#waiting[0]
    if (!first) {
      this.request = null
      return
    }
    this.request = {
      serverID: first.serverID,
      command: first.command,
      terminal: first.terminal,
      done: (ok) => {
        const answered = this.#waiting.filter((w) => w.serverID === first.serverID)
        this.#waiting = this.#waiting.filter((w) => w.serverID !== first.serverID)
        for (const w of answered) w.resolve(ok)
        this.#next()
      },
    }
  }
}

export const sudoPrompt = new SudoPrompt()
