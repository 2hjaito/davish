/** Which shell-level dialog or overlay is open. */
/** `from` starts the add form as a copy of that server. */
type ServerDialog = { mode: 'add'; from?: string } | { mode: 'edit'; id: string } | null

/** What the help dialog shows: an app's help, the shortcuts, or about. */
export type HelpTopic = { kind: 'app'; appId: string } | { kind: 'shortcuts' } | { kind: 'about' }

class Ui {
  serverDialog = $state<ServerDialog>(null)
  spotlight = $state(false)
  launchpad = $state(false)
  importDialog = $state(false)
  help = $state<HelpTopic | null>(null)
  tour = $state(false)
  /** The desktop's widgets can be moved, resized and removed. */
  widgetEdit = $state(false)
  /** Text Spotlight starts with the next time it opens. */
  spotlightSeed = $state('')

  toggleSpotlight() {
    this.spotlight = !this.spotlight
  }

  addServer() {
    this.serverDialog = { mode: 'add' }
  }

  /** The add form, filled in from a saved server (no secrets), to change what differs. */
  duplicateServer(id: string) {
    this.serverDialog = { mode: 'add', from: id }
  }

  editServer(id: string) {
    this.serverDialog = { mode: 'edit', id }
  }

  closeServerDialog() {
    this.serverDialog = null
  }
}

export const ui = new Ui()
